package core

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ICKelin/opennotr/internal/logs"
	"github.com/ICKelin/opennotr/internal/proto"
	"github.com/ICKelin/opennotr/opennotrd/plugin"
	"github.com/xtaci/smux"
)

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

	// sess manager is the model of client session
	sessMgr *SessionManager
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
		sessMgr:   GetSessionManager(),
	}
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

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

	// auth key verify
	// currently we use auth key which configured in notrd.yaml
	auth := proto.C2SAuth{}
	err := proto.ReadJSON(conn, &auth)
	if err != nil {
		logs.Error("bad request, authorize fail: %v %v", err, auth)
		return
	}

	if auth.Key != s.authKey {
		logs.Error("verify key fail")
		return
	}

	// if client without domain
	// generate random domain base on time nano
	if len(auth.Domain) <= 0 {
		auth.Domain = fmt.Sprintf("%s.%s", randomDomain(time.Now().UnixNano()), s.domain)
	}

	// select a virtual ip for client.
	// a virtual ip is the ip address which can be use in our system
	// but cannot be used by other networks
	vip, err := s.dhcp.SelectIP()
	if err != nil {
		logs.Error("dhcp select ip fail: %v", err)
		return
	}

	reply := &proto.S2CAuth{
		Vip:    vip,
		Domain: auth.Domain,
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
	// 0.0.0.0:$publicPort => $vip:$localPort
	// 1. for from address, we listen 0.0.0.0:$publicPort
	// 2. for to address, we use $vip:$localPort
	// the vip is the virtual lan ip address
	// Domain is only use for restyproxy
	for _, forward := range auth.Forward {
		for publicPort, localPort := range forward.Ports {
			item := &plugin.PluginMeta{
				Protocol:      forward.Protocol,
				From:          fmt.Sprintf("0.0.0.0:%d", publicPort),
				To:            fmt.Sprintf("%s:%s", vip, localPort),
				Domain:        auth.Domain,
				RecycleSignal: make(chan struct{}),
				Ctx:           forward.RawConfig,
			}

			err = s.pluginMgr.AddProxy(item)
			if err != nil {
				logs.Error("add proxy fail: %v", err)
				return
			}
			defer s.pluginMgr.DelProxy(item)
		}
	}

	mux, err := smux.Server(conn, nil)
	if err != nil {
		logs.Error("smux server fail:%v", err)
		return
	}

	sess := newSession(mux, vip)
	s.sessMgr.AddSession(vip, sess)
	defer s.sessMgr.DeleteSession(vip)

	rttInterval := time.NewTicker(time.Millisecond * 500)
	for range rttInterval.C {
		if mux.IsClosed() {
			logs.Info("session %v close", sess.conn.RemoteAddr().String())
			return
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
