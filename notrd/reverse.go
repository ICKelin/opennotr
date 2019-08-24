package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sync"
)

type ReverseMgr struct {
	sync.Mutex
	nginx *Nginx
}

func NewReverseMgr() *ReverseMgr {
	pool := &ReverseMgr{
		nginx: NewNginx(),
	}
	return pool
}

func (mgr *ReverseMgr) Proxy(domain, clientIP string, httpPort, httpsPort int) {
	mgr.Lock()
	mgr.Unlock()

	nginxtemp := ""
	if httpPort != 0 {
		httptemp := fmt.Sprintf(nginxHttpTemplate, domain, clientIP, httpPort)
		nginxtemp += httptemp
	}

	if httpsPort != 0 {
		httpstemp := fmt.Sprintf(nginxHttpsTemplate, domain, clientIP, httpsPort)
		nginxtemp += httpstemp
	}

	if nginxtemp != "" {
		filename := fmt.Sprintf("/etc/nginx/sites-enabled/%s", domain)
		err := mgr.nginx.Store(filename, nginxtemp)
		if err != nil {
			log.Println(err)
		}
		mgr.nginx.Reload()
	}
}

func (mgr *ReverseMgr) Stop(domain string) {
	mgr.Lock()
	defer mgr.Unlock()
	mgr.nginx.Remove(domain)
	mgr.nginx.Reload()
}

type Nginx struct{}

func NewNginx() *Nginx {
	return &Nginx{}
}

func (nginx *Nginx) Store(filename, content string) error {
	log.Println("write nginx config for: ", filename, content)
	err := ioutil.WriteFile(filename, []byte(content), 666)
	return err
}

func (nginx *Nginx) Reload() error {
	log.Println("execute nginx -s reload")
	cmd := "nginx"
	args := []string{"-s", "reload"}
	_, err := exec.Command(cmd, args...).CombinedOutput()
	return err
}

func (nginx *Nginx) Remove(domain string) error {
	log.Println("remove nginx config for: ", domain)
	return os.Remove("/etc/nginx/sites-enabled/" + domain)
}

var nginxHttpTemplate = `
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

var nginxHttpsTemplate = `
server {
	listen 443;

	server_name %s;
	ssl on;
	ssl_certificate /etc/nginx/cert/tls.crt;
	ssl_certificate_key /etc/nginx/cert/tls.key;

	location / {
		proxy_redirect off;
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_pass https://%s:%d;
	}
}

`
