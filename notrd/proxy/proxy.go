package proxy

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

const (
	// http反向代理
	PROXY_HTTP = iota

	// https反向代理
	PROXY_HTTPS

	// grpc反向代理
	PROXY_GRPC
)

type Proxy struct {
	confpath string
	cert     string
	key      string
}

func New(confpath, cert, key string) *Proxy {
	return &Proxy{
		confpath: confpath,
		cert:     cert,
		key:      key,
	}
}

// 添加代理
// 添加nginx配置文件
// domain为nginx host字段匹配
// to为客户端虚拟ip地址
func (p *Proxy) Add(http, https, grpc int, domain, vhost string) {
	template := ""
	if http > 0 {
		template += fmt.Sprintf(httpTemplate, domain, vhost, http)
	}

	if https > 0 {
		template += fmt.Sprintf(httpsTemplate, domain, p.cert, p.key, vhost, https)
	}

	if grpc > 0 {
		template += fmt.Sprintf(grpcTemplate, domain, vhost, grpc)
	}

	path := fmt.Sprintf("%s/%s_%s.conf", p.confpath, domain, vhost)

	if len(template) > 0 {
		p.writeNginxConf(path, template)
	}
}

// 移除代理
// 删除nginx配置文件
func (p *Proxy) Del(domain, vhost string) {
	path := fmt.Sprintf("%s/%s_%s.conf", p.confpath, domain, vhost)
	p.deleteNginxConf(path)
}

func (p *Proxy) writeNginxConf(path string, content string) {
	log.Println("[D] write path:", path)
	log.Println("[D] write content:", content)
	fp, err := os.Create(path)
	if err != nil {
		log.Printf("[E] create file %s fail: %v\n", path, err)
		return
	}
	defer fp.Close()

	_, err = fp.Write([]byte(content))
	if err != nil {
		log.Printf("[E] write config fail: %v\n", err)
		return
	}

	err = exec.Command("nginx", []string{"-s", "reload"}...).Run()
	if err != nil {
		log.Printf("[E] reload nginx config fail: %v\n", err)
		return
	}
}

func (p *Proxy) deleteNginxConf(path string) {
	log.Printf("removing nginx config file:%s\n", path)
	err := os.Remove(path)
	log.Println("[E] remove file fail: ", err)
}

var grpcTemplate = `
server {
	listen 880 http2;
	server_name grpc.%s;
	location / {
		proxy_redirect off;
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		grpc_pass grpc://%s:%d;
	}
}
`

var httpTemplate = `
server {
	listen 80;
	server_name %s;
	location / {
		proxy_redirect off;
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_pass http://%s:%d;
	}
}

`

var httpsTemplate = `
server {
	listen 443;

	server_name %s;
	ssl on;
	ssl_certificate %s;
	ssl_certificate_key %s;

	location / {
		proxy_redirect off;
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_pass https://%s:%d;
	}
}

`
