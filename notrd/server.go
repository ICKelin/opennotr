package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"time"

	"github.com/ICKelin/opennotr/common"
)

type ServerConfig struct {
	serverAddr string
	deviceIP   string
	tapDev     bool
	clients    []*ClientConfig
}

type Server struct {
	cfg        *ServerConfig
	serverAddr string
	deviceIP   string
	iface      *Interface
	dhcp       *DHCP
	forwards   *Forward
	reverseMgr *ReverseMgr
	sndqueue   chan *NotrClientContext
}

type NotrClientContext struct {
	conn    net.Conn
	payload []byte
}

func NewServer(cfg *ServerConfig) (*Server, error) {
	iface, err := NewInterface(&InterfaceConfig{
		ip:     cfg.deviceIP,
		gw:     cfg.deviceIP,
		mask:   defaultMask,
		tapDev: cfg.tapDev,
	})
	if err != nil {
		return nil, err
	}

	dhcp, err := NewDHCP(&DHCPConfig{
		gateway: cfg.deviceIP,
	})

	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:        cfg,
		serverAddr: cfg.serverAddr,
		deviceIP:   cfg.deviceIP,
		sndqueue:   make(chan *NotrClientContext),
		reverseMgr: NewReverseMgr(),
		iface:      iface,
		dhcp:       dhcp,
		forwards:   NewForward(),
	}

	return s, nil
}

func (s *Server) Serve() error {
	listener, err := net.Listen("tcp", s.serverAddr)
	if err != nil {
		return err
	}

	log.Println("notrd已成功启动")

	go s.readIface()
	go s.writer()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			break
		}
		go s.onConn(conn)
	}

	return nil
}

func (s *Server) onConn(conn net.Conn) {
	defer conn.Close()

	cmd, payload, err := common.Decode(conn)
	if err != nil {
		log.Println("decode auth msg fail:", err, conn.RemoteAddr().String())
		return
	}

	if cmd != common.C2S_AUTHORIZE {
		log.Println("cmd invalid")
		return
	}

	auth := &common.C2SAuthorize{}
	err = json.Unmarshal(payload, &auth)
	if err != nil {
		log.Println("invalid auth msg:", err)
		return
	}

	domain := s.getClientDomain(auth.Key)
	if domain == "" {
		log.Println("can not get client", domain, "info")
		return
	}

	ip, err := s.dhcp.SelectIP()
	if err != nil {
		log.Println("not available address")
		return
	}
	defer s.dhcp.RecycleIP(ip)

	s.reverseMgr.Proxy(domain, ip, auth.HttpPort, auth.HttpsPort)
	defer s.reverseMgr.Stop(domain)

	s2c := &common.S2CAuthorize{
		AccessIP: ip,
		Gateway:  s.deviceIP,
		Domain:   domain,
	}

	s.forwards.Add(ip, conn)
	defer s.forwards.Del(ip)

	body := common.ObjBody(s2c, nil)
	s.authResp(conn, body)

	log.Println("accept notr from", conn.RemoteAddr().String(), "assign ip", ip)
	s.reader(conn)
}

// writer write buf from send channel to net.Conn
func (s *Server) writer() {
	for ctx := range s.sndqueue {
		ctx.conn.SetWriteDeadline(time.Now().Add(time.Second * 30))
		_, err := ctx.conn.Write(ctx.payload)
		ctx.conn.SetWriteDeadline(time.Time{})
		if err != nil {
			log.Println(err)
		}
	}
}

// reader read packet from client, decode and write to net interface
func (s *Server) reader(conn net.Conn) {
	for {
		cmd, pkt, err := common.Decode(conn)
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			break
		}

		switch cmd {
		case common.C2S_HEARTBEAT:
			bytes := common.Encode(common.S2C_HEARTBEAT, nil)
			s.sndqueue <- &NotrClientContext{conn: conn, payload: bytes}

		case common.C2C_DATA:
			_, err = s.iface.Write(pkt)
			if err != nil {
				log.Println(err)
			}

		default:
			log.Println("unimplement cmd", cmd, len(pkt))
		}
	}
}

// readIface reade local net device and write to send channel
func (s *Server) readIface() {
	buff := make([]byte, 65536)
	for {
		nr, err := s.iface.Read(buff)
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			continue
		}

		ethOffset := 0

		if s.iface.IsTAP() {
			f := Frame(buff[:nr])
			if f.Invalid() {
				continue
			}

			if !f.IsIPV4() {
				// broadcast
				s.forwards.Broadcast(s.sndqueue, buff[:nr])
				continue
			}

			ethOffset = 14
		}

		p := Packet(buff[ethOffset:nr])

		if p.Invalid() {
			continue
		}

		if p.Version() != 4 {
			continue
		}

		peer := p.Dst()
		err = s.forwards.Peer(s.sndqueue, peer, buff[:nr])
		if err != nil {
			log.Println("send to ", peer, err)
			continue
		}
	}
}

func (s *Server) getClientDomain(clientKey string) string {
	for _, c := range s.cfg.clients {
		if c.AuthKey == clientKey {
			return c.Domain
		}
	}
	return ""
}

func (s *Server) authResp(conn net.Conn, body []byte) {
	resp := common.Encode(common.S2C_AUTHORIZE, body)
	conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
	conn.Write(resp)
	conn.SetWriteDeadline(time.Time{})
}
