package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ICKelin/opennotr/device"
	"github.com/ICKelin/opennotr/notrd/config"
	"github.com/ICKelin/opennotr/notrd/gateway"
	"github.com/ICKelin/opennotr/notrd/proxy"
	"github.com/ICKelin/opennotr/notrd/server"
)

func main() {
	confpath := flag.String("conf", "", "config file path")
	flag.Parse()

	cfg, err := config.Parse(*confpath)
	if err != nil {
		log.Println(err)
		return
	}

	// 初始化网卡设备
	dev, err := device.New()
	if err != nil {
		log.Println(err)
		return
	}
	defer dev.Close()

	err = dev.SetIP(cfg.GatewayConfig.Cidr, cfg.GatewayConfig.Cidr)
	if err != nil {
		log.Println(err)
		return
	}

	err = dev.SetRoute(cfg.GatewayConfig.Cidr, cfg.GatewayConfig.IP)
	if err != nil {
		log.Println(err)
		return
	}

	// 初始化网关
	gw := gateway.New(cfg.GatewayConfig.Cidr)

	// 初始化代理
	p := proxy.New(cfg.ProxyConfig.RemoteAddr)

	// 初始化域名解析配置
	var resolver *server.Resolver
	if len(cfg.ResolverConfig.EtcdEndpoints) > 0 {
		resolver, err = server.NewResolve(cfg.ResolverConfig.EtcdEndpoints)
		if err != nil {
			log.Println(err)
			return
		}
	}
	// 启动tcp server
	s := server.New(cfg.ServerConfig, gw, p, dev, resolver)
	fmt.Println(s.ListenAndServe())
}
