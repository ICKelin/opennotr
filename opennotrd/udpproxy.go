package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/ICKelin/opennotr/pkg/logs"
)

type UDPProxyItem struct {
	From          string
	To            string
	recycleSignal chan struct{}
}

type UDPProxy struct {
	mu     sync.Mutex
	routes map[string]*UDPProxyItem

	// session store udp session info
	session map[string]string
}

func NewUDPProxy() *UDPProxy {
	return &UDPProxy{
		routes: make(map[string]*UDPProxyItem),
	}
}

func (p *UDPProxy) AddProxy(from, to string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.routes[from]; ok {
		return fmt.Errorf("port %s is in used", from)
	}

	item := &UDPProxyItem{
		From:          from,
		To:            to,
		recycleSignal: make(chan struct{}),
	}

	err := p.runProxy(item)
	if err != nil {
		return err
	}

	p.routes[from] = item
	return nil
}

func (p *UDPProxy) DelProxy(from string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	item, ok := p.routes[from]
	if !ok {
		return
	}

	// send recycle port signal
	// proxy will close local connection
	select {
	case item.recycleSignal <- struct{}{}:
	default:
	}
	delete(p.routes, from)
}

func (p *UDPProxy) runProxy(item *UDPProxyItem) error {
	from := item.From
	laddr, err := net.ResolveUDPAddr("udp", from)
	if err != nil {
		return err
	}

	lis, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}
	var backendAddr *net.UDPAddr
	var buf = make([]byte, 64*1024)

	sess := make(map[string]*net.UDPConn)
	go func() {
		select {
		case <-item.recycleSignal:
			logs.Info("receive recycle signal for %s", from)
			lis.Close()
		}
	}()

	go func() {
		defer lis.Close()
		for {
			nr, raddr, err := lis.ReadFromUDP(buf)
			if err != nil {
				logs.Error("read from udp fail: %v", err)
				break
			}

			key := raddr.String()
			backendConn := sess[key]
			if backendConn == nil {
				p.mu.Lock()
				item, ok := p.routes[from]
				p.mu.Unlock()
				if !ok {
					break
				}

				backendAddr, err = net.ResolveUDPAddr("udp", item.To)
				if err != nil {
					logs.Error("resolve udp fail: %v", err)
					break
				}

				logs.Info("create new udp connection to %s", item.To)
				backendConn, err = net.DialUDP("udp", nil, backendAddr)
				if err != nil {
					logs.Error("dial udp fail: %v", err)
					break
				}
				sess[key] = backendConn

				// read from $to address and write to $from address
				go p.udpCopy(backendConn, lis, raddr)
			}
			logs.Info("write to backend %d bytes", nr)
			// read from $from address and write to $to address
			backendConn.Write(buf[:nr])
		}
	}()
	return nil
}

func (p *UDPProxy) udpCopy(src, dst *net.UDPConn, toaddr *net.UDPAddr) {
	defer src.Close()
	buf := make([]byte, 64*1024)
	for {
		nr, _, err := src.ReadFromUDP(buf)
		if err != nil {
			logs.Error("read from udp fail: %v", err)
			break
		}
		logs.Info("write back to client %d bytes", nr)
		_, err = dst.WriteToUDP(buf[:nr], toaddr)
		if err != nil {
			logs.Error("write to udp fail: %v", err)
			break
		}
	}
}
