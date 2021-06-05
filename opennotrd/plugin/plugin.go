package plugin

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ICKelin/opennotr/internal/logs"
)

var pluginMgr = &PluginManager{
	routes:  make(map[string]*PluginMeta),
	plugins: make(map[string]IPlugin),
}

// ProxyTuple defineds plugins real proxy address
type ProxyTuple struct {
	Protocol string
	FromPort string
	ToPort   string
}

// PluginMeta defineds data that the plugins needs
// these members are filled by server.go
type PluginMeta struct {
	// plugin register protocol
	// eg: tcp, udp, http, http2, h2c
	Protocol string

	// From specific local listener address of plugin
	// browser or other clients will connect to this address
	// it's no use for restyproxy plugin.
	From string

	// To specific VIP:port of our VPN peer node.
	// For example:
	// our VPN virtual lan cidr is 100.64.100.1/24
	// the connected VPN client's VPN lan ip is 100.64.100.10/24
	// and it wants to export 8080 as http port, so the $To is
	// 100.64.100.10:8080
	To string

	// Domain specific the domain of our VPN peer node.
	// It could be empty
	Domain string

	// Data you want to passto plugin
	// Reserve
	Ctx           interface{}
	RecycleSignal chan struct{}
}

func (item *PluginMeta) identify() string {
	return fmt.Sprintf("%s:%s:%s", item.Protocol, item.From, item.Domain)
}

// IPlugin defines plugin interface
// Plugin should implements the IPlugin
type IPlugin interface {
	// Setup calls at the begin of plugin system initialize
	// plugin system will pass the raw message to plugin's Setup function
	Setup(json.RawMessage) error

	// Close a proxy, it may be called by client's connection close
	StopProxy(item *PluginMeta)

	// Run a proxy, it may be called by client's connection established
	RunProxy(item *PluginMeta) (*ProxyTuple, error)
}

type PluginManager struct {
	mu sync.Mutex

	// routes stores proxier of localAddress
	// key: pluginMeta.identify()
	// value: pluginMeta
	routes map[string]*PluginMeta

	// plugins store plugin information
	// by call plugin.Register function.
	// key: protocol, eg: tcp, udp
	// value: plugin implement
	plugins map[string]IPlugin
}

func DefaultPluginManager() *PluginManager {
	return pluginMgr
}

func Register(protocol string, p IPlugin) {
	pluginMgr.plugins[protocol] = p
}

func Setup(plugins map[string]string) error {
	for protocol, cfg := range plugins {
		logs.Info("setup for %s with configuration:\n%s", protocol, cfg)
		plug, ok := pluginMgr.plugins[protocol]
		if !ok {
			logs.Error("protocol %s not register", protocol)
			return fmt.Errorf("protocol %s not register", protocol)
		}

		err := plug.Setup([]byte(cfg))
		if err != nil {
			logs.Error("setup protocol %s fail: %v", protocol, err)
			return err
		}
	}

	return nil
}

func (p *PluginManager) AddProxy(item *PluginMeta) (*ProxyTuple, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := item.identify()
	if _, ok := p.routes[key]; ok {
		return nil, fmt.Errorf("port %s is in used", key)
	}

	plug, ok := p.plugins[item.Protocol]
	if !ok {
		return nil, fmt.Errorf("proxy %s not register", item.Protocol)
	}

	tuple, err := plug.RunProxy(item)
	if err != nil {
		logs.Error("run proxy fail: %v", err)
		return nil, err
	}
	p.routes[key] = item
	return tuple, nil
}

func (p *PluginManager) DelProxy(item *PluginMeta) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := item.identify()

	plug, ok := p.plugins[item.Protocol]
	if ok {
		plug.StopProxy(item)
	}

	delete(p.routes, key)
}
