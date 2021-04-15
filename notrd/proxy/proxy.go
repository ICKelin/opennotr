package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
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

func (p *Proxy) Add(httpPort, httpsPort, grpcPort int, domain, vhost string) {
	if httpPort != 0 {
		addProxyBody := &AddProxyBody{
			Scheme: "http",
			Host:   domain,
			IP:     vhost,
			Port:   fmt.Sprintf("%d", httpPort),
		}

		p.sendReq(addProxyBody)
	}

	if httpsPort != 0 {
		addProxyBody := &AddProxyBody{
			Scheme: "https",
			Host:   domain,
			IP:     vhost,
			Port:   fmt.Sprintf("%d", httpsPort),
		}

		p.sendReq(addProxyBody)
	}

	if grpcPort != 0 {
		addProxyBody := &AddProxyBody{
			Scheme: "http2",
			Host:   domain,
			IP:     vhost,
			Port:   fmt.Sprintf("%d", grpcPort),
		}

		p.sendReq(addProxyBody)
	}
}

func (p *Proxy) Del(domain, vhost string) {
	cli := http.Client{
		Timeout: time.Second * 10,
	}
	url := fmt.Sprintf("%s?host=%s", domain)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	resp, err := cli.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	cnt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("reply from resty: %s\n", string(cnt))
}

func (p *Proxy) sendReq(body interface{}) {
	cli := http.Client{
		Timeout: time.Second * 10,
	}

	buf, _ := json.Marshal(body)
	br := bytes.NewBuffer(buf)

	req, err := http.NewRequest("POST", p.remoteAddr, br)
	if err != nil {
		fmt.Println(err)
		return
	}

	resp, err := cli.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	cnt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("reply from resty: %s\n", string(cnt))
}
