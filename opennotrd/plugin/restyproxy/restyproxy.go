package restyproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/ICKelin/opennotr/internal/logs"
	"github.com/ICKelin/opennotr/opennotrd/plugin"
)

var restyAdminUrl string

func init() {
	plugin.Register("http", &RestyProxy{})
	plugin.Register("https", &RestyProxy{})
	plugin.Register("h2c", &RestyProxy{})
}

type AddUpstreamBody struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	IP     string `json:"ip"`
	Port   string `json:"port"`
}

type RestyConfig struct {
	RestyAdminUrl string `json:"adminUrl"`
}

type RestyProxy struct {
	cfg RestyConfig
}

func (p *RestyProxy) Setup(config json.RawMessage) error {
	var cfg = RestyConfig{}
	err := json.Unmarshal([]byte(config), &cfg)
	if err != nil {
		return err
	}
	p.cfg = cfg
	return nil
}

func (p *RestyProxy) StopProxy(item *plugin.PluginMeta) {
	p.sendDeleteReq(item.Domain, item.Protocol)
}

func (p *RestyProxy) RunProxy(item *plugin.PluginMeta) (*plugin.ProxyTuple, error) {
	vip, port, err := net.SplitHostPort(item.To)
	if err != nil {
		return nil, err
	}

	req := &AddUpstreamBody{
		Scheme: item.Protocol,
		Host:   item.Domain,
		IP:     vip,
		Port:   port,
	}

	go p.sendPostReq(req)

	_, toPort, _ := net.SplitHostPort(item.To)
	return &plugin.ProxyTuple{
		Protocol: item.Protocol,
		ToPort:   toPort,
	}, nil
}

func (p *RestyProxy) sendPostReq(body interface{}) {
	cli := http.Client{
		Timeout: time.Second * 10,
	}

	buf, _ := json.Marshal(body)
	br := bytes.NewBuffer(buf)

	req, err := http.NewRequest("POST", p.cfg.RestyAdminUrl, br)
	if err != nil {
		logs.Error("request %v fail: %v", body, err)
		return
	}

	resp, err := cli.Do(req)
	if err != nil {
		logs.Error("request %v fail: %v", body, err)
		return
	}
	defer resp.Body.Close()

	cnt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Error("request %v fail: %v", body, err)
		return
	}
	logs.Info("set upstream %v reply: %s", body, string(cnt))
}

func (p *RestyProxy) sendDeleteReq(host, scheme string) {
	cli := http.Client{
		Timeout: time.Second * 10,
	}
	url := fmt.Sprintf("%s?host=%s&scheme=%s", p.cfg.RestyAdminUrl, host, scheme)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		logs.Error("delete host %s fail: %v", host, err)
		return
	}

	resp, err := cli.Do(req)
	if err != nil {
		logs.Error("delete host %s fail: %v", host, err)
		return
	}
	defer resp.Body.Close()

	cnt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Error("delete host %s fail: %v", host, err)
		return
	}

	logs.Info("delete upstream reply: %s", string(cnt))
}
