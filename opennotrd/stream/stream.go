package stream

import (
	"encoding/json"
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
	Host          string      // host for restyproxy
	Ctx           interface{} // data pass to proxier
	RecycleSignal chan struct{}
}

func (item *ProxyItem) identify() string {
	return fmt.Sprintf("%s:%s:%s:%s", item.Protocol, item.From, item.To, item.Host)
}

// Proxier defines stream proxy API
type Proxier interface {
	Setup(json.RawMessage) error
	StopProxy(item *ProxyItem)
	RunProxy(item *ProxyItem) error
}

type Stream struct {
	mu sync.Mutex

	// routes stores proxier of localAddress
	// key: proxyItem.identify()
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

func (p *Stream) Setup(plugins map[string]string) error {
	for protocol, cfg := range plugins {
		logs.Info("setup for %s with configuration:\n%s", protocol, cfg)
		proxier, ok := stream.proxier[protocol]
		if !ok {
			logs.Error("protocol %s not register", protocol)
			return fmt.Errorf("protocol %s not register", protocol)
		}

		err := proxier.Setup([]byte(cfg))
		if err != nil {
			logs.Error("setup protocol %s fail: %v", protocol, err)
			return err
		}
	}

	return nil
}

func (p *Stream) AddProxy(item *ProxyItem) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := item.identify()
	if _, ok := p.routes[key]; ok {
		return fmt.Errorf("port %s is in used", key)
	}

	proxier, ok := p.proxier[item.Protocol]
	if !ok {
		return fmt.Errorf("proxy %s not register", item.Protocol)
	}

	err := proxier.RunProxy(item)
	if err != nil {
		logs.Error("run proxy fail: %v", err)
		return err
	}
	p.routes[key] = item
	return nil
}

func (p *Stream) DelProxy(item *ProxyItem) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := item.identify()

	proxier, ok := p.proxier[item.Protocol]
	if ok {
		proxier.StopProxy(item)
	}

	delete(p.routes, key)
}
