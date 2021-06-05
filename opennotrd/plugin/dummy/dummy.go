package dummy

import (
	"encoding/json"

	"github.com/ICKelin/opennotr/internal/logs"
	"github.com/ICKelin/opennotr/opennotrd/plugin"
)

func init() {
	plugin.Register("dummy", &DummyPlugin{})
}

type DummyPlugin struct{}

func (d *DummyPlugin) Setup(cfg json.RawMessage) error {
	return nil
}

func (d *DummyPlugin) RunProxy(meta *plugin.PluginMeta) (*plugin.ProxyTuple, error) {
	logs.Info("dummy plugin client config: %v", meta.Ctx)
	return &plugin.ProxyTuple{}, nil
}

func (d *DummyPlugin) StopProxy(meta *plugin.PluginMeta) {}
