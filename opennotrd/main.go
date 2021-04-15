package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ICKelin/opennotr/device"
	"github.com/ICKelin/opennotr/opennotrd/config"
	"github.com/ICKelin/opennotr/opennotrd/server"
	"github.com/ICKelin/opennotr/pkg/logs"
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

	// create dhcp manager
	// dhcp Select/Release ip for opennotr client
	dhcp, err := server.NewDHCP(cfg.GatewayConfig.Cidr)
	if err != nil {
		logs.Error("new dhcp module fail: %v", err)
		return
	}

	// create upstream manager
	// upstream manager send http POST/DELETE to create/delete upstream
	// the api server is based on openresty
	p := server.NewUpstreamManager(cfg.ProxyConfig.RemoteAddr)

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
	s := server.New(cfg.ServerConfig, dhcp, p, dev, resolver)
	fmt.Println(s.ListenAndServe())
}
