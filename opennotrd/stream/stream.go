package stream

import (
	"fmt"
	"sync"

	"github.com/ICKelin/opennotr/pkg/logs"
)

var stream = &Stream{
	routes:  make(map[string]*ProxyItem),
	proxier: make(map[string]Proxier),
}

type ProxyItem struct {
	Protocol      string
	From          string
	To            string
	Ctx           interface{} // data pass to proxier
	recycleSignal chan struct{}
}

// Proxier defines stream proxy API
type Proxier interface {
	RunProxy(item *ProxyItem) error
}

type Stream struct {
	mu sync.Mutex

	// routes stores proxier of localAddress
	// key: protocol://localAddr eg: tcp://0.0.0.0:2222
	// value: proxyItem
	routes map[string]*ProxyItem

	// proxier stores proxier info of each registerd proxier
	// by call RegisterProxier function.
	// key: protocol, eg: tcp, udp
	// value: proxy implement
	proxier map[string]Proxier
}

func DefaultStream() *Stream {
	return stream
}

func RegisterProxier(protocol string, proxier Proxier) {
	stream.proxier[protocol] = proxier
}

func (p *Stream) AddProxy(protocol, from, to string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := protocol + "://" + from
	if _, ok := p.routes[key]; ok {
		return fmt.Errorf("port %s is in used", key)
	}

	proxier, ok := p.proxier[protocol]
	if !ok {
		return fmt.Errorf("proxy %s not register", protocol)
	}

	item := &ProxyItem{
		Protocol:      protocol,
		From:          from,
		To:            to,
		recycleSignal: make(chan struct{}),
	}

	err := proxier.RunProxy(item)
	if err != nil {
		logs.Error("run proxy fail: %v", err)
		return err
	}
	p.routes[key] = item
	return nil
}

func (p *Stream) DelProxy(protocol, from string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := protocol + "://" + from
	item, ok := p.routes[key]
	if !ok {
		return
	}

	// send recycle signal
	// proxier will close local connection
	select {
	case item.recycleSignal <- struct{}{}:
	default:
	}
	delete(p.routes, key)
}
