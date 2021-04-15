package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/ICKelin/opennotr/pkg/device"
	"github.com/ICKelin/opennotr/pkg/proto"
)

type writeReq struct {
	cmd  int
	data []byte
}

type Client struct {
	srv    string
	key    string
	domain string
	http   int
	https  int
	grpc   int
}

func NewClient(cfg *Config) *Client {
	return &Client{
		srv:    cfg.ServerAddr,
		key:    cfg.Key,
		domain: cfg.Domain,
		http:   cfg.HTTP,
		https:  cfg.HTTPS,
		grpc:   cfg.Grpc,
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
			Key:    c.key,
			Domain: c.domain,
			HTTP:   c.http,
			HTTPS:  c.https,
			Grpc:   c.grpc,
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

		writebuf := make(chan *writeReq)

		dev, err := device.New()
		if err != nil {
			log.Println(err)
			return
		}

		err = dev.SetIP(auth.Gateway, auth.Vip)
		if err != nil {
			log.Println(err)
			return
		}

		err = dev.SetRoute(auth.Gateway, auth.Vip)
		if err != nil {
			log.Println(err)
			return
		}

		go c.readDev(dev, writebuf)

		ctx, cancel := context.WithCancel(context.Background())
		go c.writer(ctx, conn, writebuf)
		c.reader(dev, conn, writebuf)
		cancel()
		dev.Close()

		log.Println("reconnecting")
		time.Sleep(time.Second * 3)
	}
}

func (c *Client) reader(dev *device.Device, conn net.Conn, writeBuf chan *writeReq) {
	defer conn.Close()

	for {
		hdr, body, err := proto.Read(conn)
		if err != nil {
			log.Println(err)
			break
		}

		switch hdr.Cmd() {
		case proto.CmdHeartbeat:
			proto.Write(conn, proto.CmdHeartbeat, nil)

		case proto.CmdData:
			dev.Write(body)

		case proto.CmdAuth:
			log.Println("authorize return: ", string(body))

		default:
		}
	}
}

func (c *Client) writer(ctx context.Context, conn net.Conn, writeBuf chan *writeReq) {
	defer conn.Close()

	for {
		select {
		case msg := <-writeBuf:
			err := proto.Write(conn, msg.cmd, msg.data)
			if err != nil {
				log.Println("write fail: ", err)
				return
			}

		case <-ctx.Done():
			log.Println("close writer")
			return
		}
	}
}

func (c *Client) readDev(dev *device.Device, writeBuf chan *writeReq) {
	for {
		bytes, err := dev.Read()
		if err != nil {
			log.Println("read device fail: ", err)
			break
		}

		req := &writeReq{
			cmd:  proto.CmdData,
			data: bytes,
		}

		select {
		case writeBuf <- req:
		default:
		}
	}
}
