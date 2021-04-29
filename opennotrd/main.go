package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ICKelin/opennotr/opennotrd/stream"
	"github.com/ICKelin/opennotr/pkg/device"
	"github.com/ICKelin/opennotr/pkg/logs"
)

func main() {
	confpath := flag.String("conf", "", "config file path")
	flag.Parse()

	cfg, err := ParseConfig(*confpath)
	if err != nil {
		fmt.Println(err)
		return
	}

	logs.Init("opennotrd.log", "debug", 10)

	// initial tun device
	dev, err := device.New()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer dev.Close()

	err = dev.SetIP(cfg.DHCPConfig.Cidr, cfg.DHCPConfig.Cidr)
	if err != nil {
		log.Println(err)
		return
	}

	err = dev.SetRoute(cfg.DHCPConfig.Cidr, cfg.DHCPConfig.IP)
	if err != nil {
		log.Println(err)
		return
	}

	// create dhcp manager
	// dhcp Select/Release ip for opennotr client
	dhcp, err := NewDHCP(cfg.DHCPConfig.Cidr)
	if err != nil {
		logs.Error("new dhcp module fail: %v", err)
		return
	}

	// create upstream manager
	// upstream manager send http POST/DELETE to create/delete upstream
	// the api server is based on openresty
	// p := NewUpstreamManager(cfg.UpstreamConfig.RemoteAddr)
	stream.SetRestyAdminUrl(cfg.UpstreamConfig.RemoteAddr)

	// 初始化域名解析配置
	var resolver *Resolver
	if len(cfg.ResolverConfig.EtcdEndpoints) > 0 {
		resolver, err = NewResolve(cfg.ResolverConfig.EtcdEndpoints)
		if err != nil {
			log.Println(err)
			return
		}
	}
	// 启动tcp server
	s := NewServer(cfg.ServerConfig, dhcp, dev, resolver)
	fmt.Println(s.ListenAndServe())
}
