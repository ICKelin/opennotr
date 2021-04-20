package main

import (
	"fmt"
	"sync"

	"github.com/inetaf/tcpproxy"
)

type TCPProxyItem struct {
	LocalPort  int
	RemoteIP   string
	RemotePort int
}

type TCPProxy struct {
	mu     sync.Mutex
	routes map[string]*tcpproxy.Proxy
}

func NewTCPProxy() *TCPProxy {
	return &TCPProxy{
		routes: make(map[string]*tcpproxy.Proxy),
	}
}

func (t *TCPProxy) AddProxy(from, to string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.routes[from]
	if ok {
		return fmt.Errorf("port %s is in used", from)
	}

	p := &tcpproxy.Proxy{}
	t.routes[from] = p

	target := tcpproxy.To(to)
	p.AddRoute(from, target)
	go p.Run()
	return nil
}

func (t *TCPProxy) DelProxy(from string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	p := t.routes[from]
	if p != nil {
		p.Close()
		delete(t.routes, from)
	}
}
