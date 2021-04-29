package stream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/ICKelin/opennotr/pkg/logs"
)

var restyAdminUrl string

func init() {
	RegisterProxier("http", &RestyProxy{})
	RegisterProxier("https", &RestyProxy{})
	RegisterProxier("h2c", &RestyProxy{})
}

type AddUpstreamBody struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	IP     string `json:"ip"`
	Port   string `json:"port"`
}

type RestyProxy struct{}

func SetRestyAdminUrl(url string) {
	restyAdminUrl = url
}

func (p *RestyProxy) Setup(config json.RawMessage) {}

func (p *RestyProxy) StopProxy(item *ProxyItem) {
	p.sendDeleteReq(item.Host, item.Protocol)
}

func (p *RestyProxy) RunProxy(item *ProxyItem) error {
	vip, port, err := net.SplitHostPort(item.To)
	if err != nil {
		return err
	}

	req := &AddUpstreamBody{
		Scheme: item.Protocol,
		Host:   item.Host,
		IP:     vip,
		Port:   port,
	}

	go p.sendPostReq(req)
	return nil
}

func (p *RestyProxy) sendPostReq(body interface{}) {
	cli := http.Client{
		Timeout: time.Second * 10,
	}

	buf, _ := json.Marshal(body)
	br := bytes.NewBuffer(buf)

	req, err := http.NewRequest("POST", restyAdminUrl, br)
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
	url := fmt.Sprintf("%s?host=%s&scheme=%s", restyAdminUrl, host, scheme)
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
