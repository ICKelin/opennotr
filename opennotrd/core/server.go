package core

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ICKelin/opennotr/opennotrd/plugin"
	"github.com/ICKelin/opennotr/pkg/logs"
	"github.com/ICKelin/opennotr/pkg/proto"
	"github.com/hashicorp/yamux"
)

type Session struct {
	conn       *yamux.Session
	clientAddr string
	rxbytes    uint64
	txbytes    uint64
}

func newSession(conn *yamux.Session, clientAddr string) *Session {
	return &Session{
		conn:       conn,
		clientAddr: clientAddr,
	}
}

type Server struct {
	cfg      ServerConfig
	addr     string
	authKey  string
	domain   string
	publicIP string

	// dhcp manager select/release ip for client
	dhcp *DHCP

	// call stream proxy for dynamic add/del tcp/udp proxy
	pluginMgr *plugin.PluginManager

	// resolver writes domains to etcd and it will be used by coredns
	resolver *Resolver

	// sess store client connect wraper
	// key: client virtual ip(vip)
	// value: *Session
	sess sync.Map
}

func NewServer(cfg ServerConfig,
	dhcp *DHCP,
	resolver *Resolver) *Server {
	return &Server{
		cfg:       cfg,
		addr:      cfg.ListenAddr,
		authKey:   cfg.AuthKey,
		domain:    cfg.Domain,
		publicIP:  publicIP(),
		dhcp:      dhcp,
		pluginMgr: plugin.DefaultPluginManager(),
		resolver:  resolver,
	}
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	go s.tproxyTCP(s.cfg.TCPProxyListen)
	go s.tproxyUDP(s.cfg.UDPProxyListen)

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
		return
	}

	reply := &proto.S2CAuth{
		Vip:     vip,
		Gateway: s.dhcp.GetCIDR(),
		Domain:  auth.Domain,
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

			err := s.pluginMgr.AddProxy(item)
			if err != nil {
				logs.Error("add proxy fail: %v", err)
				return
			}
			defer s.pluginMgr.DelProxy(item)
		}
	}

	mux, err := yamux.Server(conn, nil)
	if err != nil {
		logs.Error("yamux server fail:%v", err)
		return
	}

	// tunnel session
	sess := newSession(mux, conn.RemoteAddr().String())
	s.sess.Store(vip, sess)
	defer s.sess.Delete(vip)

	rttInterval := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-mux.CloseChan():
			logs.Info("session %v close", sess.conn.RemoteAddr().String())
			return
		case <-rttInterval.C:
			rx := atomic.SwapUint64(&sess.rxbytes, 0)
			tx := atomic.SwapUint64(&sess.txbytes, 0)
			rtt, _ := mux.Ping()
			logs.Debug("session %s rtt %d, rx %d tx %d",
				sess.conn.RemoteAddr().String(), rtt.Milliseconds(), rx, tx)
		}
	}
}

func (s *Server) tproxyTCP(listenAddr string) error {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	// set socket with ip transparent option
	file, err := listener.(*net.TCPListener).File()
	if err != nil {
		return err
	}
	defer file.Close()

	err = syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			logs.Error("accept fail: %v", err)
			break
		}

		go s.tcpProxy(conn)
	}

	return nil
}

func (s *Server) tcpProxy(conn net.Conn) {
	dip, dport, _ := net.SplitHostPort(conn.LocalAddr().String())
	sip, sport, _ := net.SplitHostPort(conn.RemoteAddr().String())

	val, ok := s.sess.Load(dip)
	if !ok {
		logs.Error("no route to host: %s", dip)
		conn.Close()
		return
	}

	stream, err := val.(*Session).conn.OpenStream()
	if err != nil {
		logs.Error("open stream fail: %v", err)
		conn.Close()
		return
	}

	buf := make([]byte, 2)

	//  write proxy protocol packet
	proxyProtocol := &proto.ProxyProtocol{
		Protocol: "tcp",
		SrcIP:    sip,
		SrcPort:  sport,
		// DstIP:    dip,
		DstIP:   "127.0.0.1", // may change to client setting
		DstPort: dport,
	}

	body, err := json.Marshal(proxyProtocol)
	if err != nil {
		logs.Error("json marshal fail: %v", err)
		conn.Close()
		stream.Close()
		return
	}

	binary.BigEndian.PutUint16(buf, uint16(len(body)))
	buf = append(buf, body...)
	stream.SetWriteDeadline(time.Now().Add(time.Second * 10))
	_, err = stream.Write(buf)
	stream.SetWriteDeadline(time.Time{})
	if err != nil {
		logs.Error("stream write fail: %v", err)
		conn.Close()
		stream.Close()
		return
	}

	go func() {
		defer stream.Close()
		defer conn.Close()
		io.Copy(stream, conn)
	}()

	go func() {
		defer stream.Close()
		defer conn.Close()
		io.Copy(conn, stream)
	}()
}

func (s *Server) tproxyUDP(listenAddr string) {}

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
