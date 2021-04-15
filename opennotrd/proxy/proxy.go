package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ICKelin/opennotr/pkg/logs"
)

type Proxy struct {
	remoteAddr string
}

type AddProxyBody struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	IP     string `json:"ip"`
	Port   string `json:"port"`
}

func New(remoteAddr string) *Proxy {
	return &Proxy{
		remoteAddr: remoteAddr,
	}
}

func (p *Proxy) Add(httpPort, httpsPort, grpcPort int, domain, vip string) {
	if httpPort != 0 {
		addProxyBody := &AddProxyBody{
			Scheme: "http",
			Host:   domain,
			IP:     vhost,
			Port:   fmt.Sprintf("%d", httpPort),
		}

		p.sendPostReq(addProxyBody)
	}

	if httpsPort != 0 {
		addProxyBody := &AddProxyBody{
			Scheme: "https",
			Host:   domain,
			IP:     vhost,
			Port:   fmt.Sprintf("%d", httpsPort),
		}

		p.sendPostReq(addProxyBody)
	}

	if grpcPort != 0 {
		addProxyBody := &AddProxyBody{
			Scheme: "http2",
			Host:   domain,
			IP:     vhost,
			Port:   fmt.Sprintf("%d", grpcPort),
		}

		p.sendPostReq(addProxyBody)
	}
}

func (p *Proxy) Del(host string, httpPort, httpsPort, grpcPort int) {
	if httpPort != 0 {
		p.sendDeleteReq(host, "http")
	}

	if httpsPort != 0 {
		p.sendDeleteReq(host, "https")
	}

	if grpcPort != 0 {
		p.sendDeleteReq(host, "http2")
	}
}

func (p *Proxy) sendPostReq(body interface{}) {
	cli := http.Client{
		Timeout: time.Second * 10,
	}

	buf, _ := json.Marshal(body)
	br := bytes.NewBuffer(buf)

	req, err := http.NewRequest("POST", p.remoteAddr, br)
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
	logs.Info("reply from resty: %s", string(cnt))
}

func (p *Proxy) sendDeleteReq(host, scheme string) {
	cli := http.Client{
		Timeout: time.Second * 10,
	}
	url := fmt.Sprintf("%s?host=%s&scheme=%s", p.remoteAddr, host, scheme)
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

	logs.Info("reply from resty: %s", string(cnt))
}
