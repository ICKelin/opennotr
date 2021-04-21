package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ICKelin/opennotr/pkg/logs"
)

type UpstreamManager struct {
	remoteAddr string
}

type AddUpstreamBody struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	IP     string `json:"ip"`
	Port   string `json:"port"`
}

func NewUpstreamManager(remoteAddr string) *UpstreamManager {
	return &UpstreamManager{
		remoteAddr: remoteAddr,
	}
}

// Create upstream for $host, backend $vip
func (p *UpstreamManager) AddUpstream(httpPort, httpsPort, grpcPort int, host, vip string) {
	if httpPort != 0 {
		addProxyBody := &AddUpstreamBody{
			Scheme: "http",
			Host:   host,
			IP:     vip,
			Port:   fmt.Sprintf("%d", httpPort),
		}

		p.sendPostReq(addProxyBody)
	}

	if httpsPort != 0 {
		addProxyBody := &AddUpstreamBody{
			Scheme: "https",
			Host:   host,
			IP:     vip,
			Port:   fmt.Sprintf("%d", httpsPort),
		}

		p.sendPostReq(addProxyBody)
	}

	if grpcPort != 0 {
		addProxyBody := &AddUpstreamBody{
			Scheme: "h2c",
			Host:   host,
			IP:     vip,
			Port:   fmt.Sprintf("%d", grpcPort),
		}

		p.sendPostReq(addProxyBody)
	}
}

func (p *UpstreamManager) DelUpstream(host string, httpPort, httpsPort, grpcPort int) {
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

func (p *UpstreamManager) sendPostReq(body interface{}) {
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
	logs.Info("set upstream reply: %s", string(cnt))
}

func (p *UpstreamManager) sendDeleteReq(host, scheme string) {
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

	logs.Info("delete upstream reply: %s", string(cnt))
}
