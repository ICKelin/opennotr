package dummy

import (
	"encoding/json"

	"github.com/ICKelin/opennotr/opennotrd/plugin"
	"github.com/ICKelin/opennotr/pkg/logs"
)

func init() {
	plugin.Register("dummy", &DummyPlugin{})
}

type DummyPlugin struct{}

func (d *DummyPlugin) Setup(cfg json.RawMessage) error {
	return nil
}

func (d *DummyPlugin) RunProxy(meta *plugin.PluginMeta) error {
	logs.Info("dummy plugin client config: %v", meta.Ctx)
	return nil
}

func (d *DummyPlugin) StopProxy(meta *plugin.PluginMeta) {}
