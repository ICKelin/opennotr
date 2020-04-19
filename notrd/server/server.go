package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ICKelin/opennotr/device"
	"github.com/ICKelin/opennotr/notrd/config"
	"github.com/ICKelin/opennotr/notrd/gateway"
	"github.com/ICKelin/opennotr/notrd/proxy"
	"github.com/ICKelin/opennotr/proto"
)

type Server struct {
	addr     string
	authKey  string
	domain   string
	publicIP string

	// gateway模块
	// 用户ip地址分配
	gw *gateway.Gateway

	// proxy模块
	// 用户管理客户端代理
	p *proxy.Proxy

	// dev模块
	// 读写网卡设备
	dev *device.Device

	// 域名解析模块
	// 设置域名解析记录
	resolver *Resolver

	// 所有客户端会话
	sess sync.Map
}

func New(cfg config.ServerConfig,
	gw *gateway.Gateway,
	p *proxy.Proxy,
	dev *device.Device,
	resolver *Resolver) *Server {
	return &Server{
		addr:     cfg.ListenAddr,
		authKey:  cfg.AuthKey,
		domain:   cfg.Domain,
		publicIP: publicIP(),
		gw:       gw,
		p:        p,
		dev:      dev,
		resolver: resolver,
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
		log.Println("bad request, authorize fail: ", err)
		return
	}

	if auth.Key != s.authKey {
		log.Println("verify key fail")
		return
	}

	if len(auth.Domain) <= 0 {
		auth.Domain = fmt.Sprintf("%s.%s", randomDomain(time.Now().Unix()), s.domain)
	}

	// proxy
	vip, err := s.gw.SelectIP()
	if err != nil {
		log.Println(err)
		return
	}

	reply := &proto.S2CAuth{
		Vip:     vip,
		Gateway: s.gw.GetAddr(),
		Domain:  auth.Domain,
	}

	err = proto.WriteJSON(conn, proto.CmdAuth, reply)
	if err != nil {
		log.Println(err)
		return
	}

	// 如果使用内网模式，域名解析为vip，需要启动客户端才能访问，安全性较好
	// 如果使用公网模式，域名解析为public_ip公网地址，通过公网服务器即可访问，安全性较差
	// 目前只支持公网模式
	err = s.resolver.ApplyDomain(auth.Domain, publicIP())
	if err != nil {
		log.Printf("resolve domain fail: %v\n", err)
		return
	}

	s.p.Add(auth.HTTP, auth.HTTPS, auth.Grpc, auth.Domain, vip)
	defer s.p.Del(auth.Domain, vip)

	log.Println("select vip:", vip)
	log.Println("select domain:", auth.Domain)

	// tunnel
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

// 读客户端
func (s *Server) reader(ctx context.Context, sess *Session, finread chan struct{}) {
	defer close(finread)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 解码
		sess.conn.SetReadDeadline(time.Now().Add(time.Second * 30))
		hdr, body, err := proto.Read(sess.conn)
		sess.conn.SetReadDeadline(time.Time{})
		if err != nil {
			log.Println(err)
			break
		}

		switch hdr.Cmd() {
		case proto.CmdHeartbeat:
			atomic.AddInt32(&sess.activePing, -1)

		case proto.CmdData:
			s.dev.Write(body)

		default:
			log.Println("unsupported cmd: ", hdr.Cmd(), body)
		}
	}
}

// 写客户端
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

// 心跳保活
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
			log.Println("server ping timeout")
			break
		}

		sess.hbbuf <- struct{}{}
		atomic.AddInt32(&sess.activePing, 1)
	}
}

// 读取设备数据
func (s *Server) readIface() {
	for {
		pkt, err := s.dev.Read()
		if err != nil {
			log.Println(err)
			// server的设备网卡有问题，直接退出重新拉起
			// 避免程序进入隧道联通，但是隧道不通的场景
			os.Exit(0)
		}

		v4Pkt := Packet(pkt)

		// 路由转发只支持ipv4
		if v4Pkt.Version() != 4 {
			continue
		}

		log.Printf("[D] src %s dst %s\n", v4Pkt.Src(), v4Pkt.Dst())

		obj, ok := s.sess.Load(v4Pkt.Dst())
		if !ok {
			log.Printf("vip %s not found\n", v4Pkt.Dst())
			continue
		}

		select {
		case obj.(*Session).writebuf <- pkt:
		default:
		}
	}
}

// 生产随机域名
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

// 获取公网ip
// 获取不到，程序启动失败
func publicIP() string {
	resp, err := http.Get("http://ipv4.icanhazip.com")
	if err != nil {
		log.Println(err)
		os.Exit(0)
		return ""
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		os.Exit(0)
		return ""
	}

	str := string(content)
	idx := strings.LastIndex(str, "\n")
	return str[:idx]
}
