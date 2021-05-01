package plugin

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ICKelin/opennotr/pkg/logs"
)

var pluginMgr = &PluginManager{
	routes:  make(map[string]*PluginMeta),
	plugins: make(map[string]IPlugin),
}

type PluginMeta struct {
	Protocol      string
	From          string
	To            string
	Domain        string
	Ctx           interface{} // data pass to plugin
	RecycleSignal chan struct{}
}

func (item *PluginMeta) identify() string {
	return fmt.Sprintf("%s:%s:%s", item.Protocol, item.From, item.Domain)
}

// IPlugin defines proxy plugin API
type IPlugin interface {
	Setup(json.RawMessage) error
	StopProxy(item *PluginMeta)
	RunProxy(item *PluginMeta) error
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

func (p *PluginManager) AddProxy(item *PluginMeta) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := item.identify()
	if _, ok := p.routes[key]; ok {
		return fmt.Errorf("port %s is in used", key)
	}

	plug, ok := p.plugins[item.Protocol]
	if !ok {
		return fmt.Errorf("proxy %s not register", item.Protocol)
	}

	err := plug.RunProxy(item)
	if err != nil {
		logs.Error("run proxy fail: %v", err)
		return err
	}
	p.routes[key] = item
	return nil
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
