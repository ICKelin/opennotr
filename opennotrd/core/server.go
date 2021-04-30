package core

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ICKelin/opennotr/opennotrd/plugin"
	"github.com/ICKelin/opennotr/pkg/device"
	"github.com/ICKelin/opennotr/pkg/logs"
	"github.com/ICKelin/opennotr/pkg/proto"
)

type Session struct {
	conn       net.Conn
	clientAddr string
	activePing int32
	hbbuf      chan struct{}
	writebuf   chan []byte
	readbuf    chan []byte
}

func newSession(conn net.Conn, clientAddr string) *Session {
	return &Session{
		conn:       conn,
		clientAddr: clientAddr,
		activePing: 0,
		hbbuf:      make(chan struct{}),
		writebuf:   make(chan []byte),
		readbuf:    make(chan []byte),
	}
}

type Server struct {
	addr     string
	authKey  string
	domain   string
	publicIP string

	// dhcp manager select/release ip for client
	dhcp *DHCP

	// call stream proxy for dynamic add/del tcp/udp proxy
	pluginMgr *plugin.PluginManager

	// tun device wraper
	dev *device.Device

	// resolver writes domains to etcd and it will be used by coredns
	resolver *Resolver

	// sess store client connect wraper
	// key: client virtual ip(vip)
	// value: *Session
	sess sync.Map
}

func NewServer(cfg ServerConfig,
	dhcp *DHCP,
	dev *device.Device,
	resolver *Resolver) *Server {
	return &Server{
		addr:      cfg.ListenAddr,
		authKey:   cfg.AuthKey,
		domain:    cfg.Domain,
		publicIP:  publicIP(),
		dhcp:      dhcp,
		pluginMgr: plugin.DefaultPluginManager(),
		dev:       dev,
		resolver:  resolver,
	}
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	go s.readIface()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go s.onConn(conn)
	}
}

func (s *Server) onConn(conn net.Conn) {
	defer conn.Close()

	// authorize
	auth := proto.C2SAuth{}
	err := proto.ReadJSON(conn, &auth)
	if err != nil {
		logs.Error("bad request, authorize fail: %v", err)
		return
	}

	if auth.Key != s.authKey {
		logs.Error("verify key fail")
		return
	}

	if len(auth.Domain) <= 0 {
		auth.Domain = fmt.Sprintf("%s.%s", randomDomain(time.Now().Unix()), s.domain)
	}

	vip, err := s.dhcp.SelectIP()
	if err != nil {
		logs.Error("dhcp select ip fail: %v", err)
		// todo reply fail
		return
	}

	reply := &proto.S2CAuth{
		Vip:     vip,
		Gateway: s.dhcp.GetCIDR(),
		Domain:  auth.Domain,
		// todo: add proxy item
	}

	err = proto.WriteJSON(conn, proto.CmdAuth, reply)
	if err != nil {
		logs.Error("write json fail: %v", err)
		return
	}

	// dynamic dns, write domain=>ip map to etcd
	// coredns will read records from etcd and reply to dns client
	if s.resolver != nil {
		err = s.resolver.ApplyDomain(auth.Domain, publicIP())
		if err != nil {
			logs.Error("resolve domain fail: %v", err)
			return
		}
	}

	logs.Info("select vip: %s", vip)
	logs.Info("select domain: %s", auth.Domain)

	// create forward
	// $localPort => vip:$upstreamPort
	// 1. for from address, we listen 0.0.0.0:$inport
	// from member is not used for restyproxy
	// 2. for to address, we use $vip:$upstreamPort
	// the vip is the virtual lan ip address
	// Domain is only use for restyproxy
	for _, forward := range auth.Forward {
		for localPort, upstreamPort := range forward.Ports {
			item := &plugin.PluginMeta{
				Protocol:      forward.Protocol,
				From:          fmt.Sprintf("0.0.0.0:%d", localPort),
				To:            fmt.Sprintf("%s:%d", vip, upstreamPort),
				Domain:        auth.Domain,
				RecycleSignal: make(chan struct{}),
			}

			s.pluginMgr.AddProxy(item)
			defer s.pluginMgr.DelProxy(item)
		}
	}

	// tunnel session
	sess := newSession(conn, conn.RemoteAddr().String())
	s.sess.Store(vip, sess)
	defer s.sess.Delete(vip)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	finread := make(chan struct{})

	go s.reader(ctx, sess, finread)
	go s.writer(ctx, sess)
	s.heartbeat(ctx, sess, finread)
}

// reader reads from session
// once error occurs, close finread channel to stop heartbeat
func (s *Server) reader(ctx context.Context, sess *Session, finread chan struct{}) {
	defer close(finread)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		sess.conn.SetReadDeadline(time.Now().Add(time.Second * 30))
		hdr, body, err := proto.Read(sess.conn)
		sess.conn.SetReadDeadline(time.Time{})
		if err != nil {
			logs.Error("read fail: %v", err)
			break
		}

		switch hdr.Cmd() {
		case proto.CmdHeartbeat:
			atomic.AddInt32(&sess.activePing, -1)

		case proto.CmdData:
			s.dev.Write(body)

		default:
			logs.Error("unsupported cmd: %d %v", hdr.Cmd(), body)
		}
	}
}

// writer writes data,heartbeat to session
func (s *Server) writer(ctx context.Context, sess *Session) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-sess.hbbuf:
			sess.conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
			proto.Write(sess.conn, proto.CmdHeartbeat, nil)
			sess.conn.SetWriteDeadline(time.Time{})

		case frame := <-sess.writebuf:
			sess.conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
			proto.Write(sess.conn, proto.CmdData, frame)
			sess.conn.SetWriteDeadline(time.Time{})
		}
	}
}

// heartbeat sends heartbeat packet to client and incr activePing by one
func (s *Server) heartbeat(ctx context.Context, sess *Session, finread chan struct{}) {
	tick := time.NewTicker(time.Second * 10)
	defer tick.Stop()

	for range tick.C {
		select {
		case <-finread:
			return
		default:
		}

		if atomic.LoadInt32(&sess.activePing) >= 3 {
			logs.Error("server ping timeout")
			break
		}

		sess.hbbuf <- struct{}{}
		atomic.AddInt32(&sess.activePing, 1)
	}
}

// readIface
func (s *Server) readIface() {
	for {
		pkt, err := s.dev.Read()
		if err != nil {
			logs.Error("read device fail: %v", err)
			break
		}

		v4Pkt := Packet(pkt)

		if v4Pkt.Version() != 4 {
			logs.Warn("not support ip version %d", v4Pkt.Version())
			continue
		}

		obj, ok := s.sess.Load(v4Pkt.Dst())
		if !ok {
			logs.Warn("vip %s not found %v", v4Pkt.Dst())
			continue
		}

		select {
		case obj.(*Session).writebuf <- pkt:
		default:
		}
	}
}

// randomDomain generate random domain for client
func randomDomain(num int64) string {
	const ALPHABET = "123456789abcdefghijklmnopqrstuvwxyz"
	const BASE = int64(len(ALPHABET))
	rs := ""
	for num > 0 {
		rs += string(ALPHABET[num%BASE])
		num = num / BASE
	}

	return rs
}

// get public
func publicIP() string {
	resp, err := http.Get("http://ipv4.icanhazip.com")
	if err != nil {
		logs.Error("get public ip fail: %v", err)
		panic(err)
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	str := string(content)
	idx := strings.LastIndex(str, "\n")
	return str[:idx]
}
