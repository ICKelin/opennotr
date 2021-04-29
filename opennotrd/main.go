package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ICKelin/opennotr/opennotrd/plugin"
	"github.com/ICKelin/opennotr/pkg/device"
	"github.com/ICKelin/opennotr/pkg/logs"

	// plugin import to register
	_ "github.com/ICKelin/opennotr/opennotrd/plugin/restyproxy"
	_ "github.com/ICKelin/opennotr/opennotrd/plugin/tcpproxy"
	_ "github.com/ICKelin/opennotr/opennotrd/plugin/udpproxy"
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
	logs.Info("config: %v", cfg)

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

	plugin.Setup(cfg.Plugins)

	// initial resolver
	// currently resolver use coredns and etcd
	// our resolver just write DOMAIN => VIP record to etcd
	var resolver *Resolver
	if len(cfg.ResolverConfig.EtcdEndpoints) > 0 {
		resolver, err = NewResolve(cfg.ResolverConfig.EtcdEndpoints)
		if err != nil {
			log.Println(err)
			return
		}
	}

	// run tunnel tcp server, it will cause tcp over tcp problems
	// it may changed to udp later.
	s := NewServer(cfg.ServerConfig, dhcp, dev, resolver)
	fmt.Println(s.ListenAndServe())
}
