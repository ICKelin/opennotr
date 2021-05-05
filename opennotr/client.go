package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/ICKelin/opennotr/pkg/logs"
	"github.com/ICKelin/opennotr/pkg/proto"
	"github.com/hashicorp/yamux"
)

type Client struct {
	srv      string
	key      string
	domain   string
	forwards []proto.ForwardItem
}

func NewClient(cfg *Config) *Client {
	return &Client{
		srv:      cfg.ServerAddr,
		key:      cfg.Key,
		domain:   cfg.Domain,
		forwards: cfg.Forwards,
	}
}

func (c *Client) Run() {
	for {
		conn, err := net.Dial("tcp", c.srv)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second * 3)
			continue
		}

		c2sauth := &proto.C2SAuth{
			Key:     c.key,
			Domain:  c.domain,
			Forward: c.forwards,
		}

		err = proto.WriteJSON(conn, proto.CmdAuth, c2sauth)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second * 3)
			continue
		}

		auth := proto.S2CAuth{}
		err = proto.ReadJSON(conn, &auth)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second * 3)
			continue
		}

		log.Println("connect success")
		log.Println("vhost:", auth.Vip)
		log.Println("domain:", auth.Domain)

		mux, err := yamux.Client(conn, nil)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second * 3)
			continue
		}

		for {
			stream, err := mux.AcceptStream()
			if err != nil {
				log.Println(err)
				break
			}

			go c.handleStream(stream)
		}

		log.Println("reconnecting")
		time.Sleep(time.Second * 3)
	}
}

func (c *Client) handleStream(stream *yamux.Stream) {
	lenbuf := make([]byte, 2)
	_, err := stream.Read(lenbuf)
	if err != nil {
		log.Println(err)
		stream.Close()
		return
	}

	bodylen := binary.BigEndian.Uint16(lenbuf)
	buf := make([]byte, bodylen)
	nr, err := io.ReadFull(stream, buf)
	if err != nil {
		log.Println(err)
		stream.Close()
		return
	}

	proxyProtocol := proto.ProxyProtocol{}
	err = json.Unmarshal(buf[:nr], &proxyProtocol)
	if err != nil {
		log.Println("unmarshal fail: ", err)
		return
	}

	switch proxyProtocol.Protocol {
	case "tcp":
		c.tcpProxy(stream, &proxyProtocol)
	case "udp":
		c.udpProxy(stream, &proxyProtocol)
	}
}

func (c *Client) tcpProxy(stream *yamux.Stream, p *proto.ProxyProtocol) {
	addr := fmt.Sprintf("%s:%s", p.DstIP, p.DstPort)
	remoteConn, err := net.DialTimeout("tcp", addr, time.Second*10)
	if err != nil {
		log.Println(err)
		stream.Close()
		return
	}

	go func() {
		defer remoteConn.Close()
		defer stream.Close()
		io.Copy(remoteConn, stream)
	}()

	go func() {
		defer remoteConn.Close()
		defer stream.Close()
		io.Copy(stream, remoteConn)
	}()
}

func (c *Client) udpProxy(stream *yamux.Stream, p *proto.ProxyProtocol) {
	addr := fmt.Sprintf("%s:%s", p.DstIP, p.DstPort)
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Println(err)
		stream.Close()
		return
	}

	remoteConn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		defer remoteConn.Close()
		defer stream.Close()
		hdr := make([]byte, 2)
		for {
			_, err := io.ReadFull(stream, hdr)
			if err != nil {
				logs.Error("read stream fail %v", err)
				break
			}
			nlen := binary.BigEndian.Uint16(hdr)
			buf := make([]byte, nlen)
			_, err = io.ReadFull(stream, buf)
			if err != nil {
				logs.Error("read stream body fail: %v", err)
				break
			}

			remoteConn.Write(buf)
		}
	}()

	go func() {
		defer remoteConn.Close()
		defer stream.Close()
		buf := make([]byte, 64*1024)
		for {
			nr, err := remoteConn.Read(buf)
			if err != nil {
				log.Println(err)
				break
			}

			bytes := encode(buf[:nr])
			_, err = stream.Write(bytes)
			if err != nil {
				log.Println(err)
				break
			}
		}
	}()
}

func encode(raw []byte) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(raw)))
	buf = append(buf, raw...)
	return buf
}
