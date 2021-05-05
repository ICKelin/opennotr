package opennotrd

import (
	"flag"
	"fmt"
	"log"

	"github.com/ICKelin/opennotr/opennotrd/core"
	"github.com/ICKelin/opennotr/opennotrd/plugin"
	"github.com/ICKelin/opennotr/pkg/logs"
)

func Run() {
	confpath := flag.String("conf", "", "config file path")
	flag.Parse()

	cfg, err := core.ParseConfig(*confpath)
	if err != nil {
		fmt.Println(err)
		return
	}

	logs.Init("opennotrd.log", "debug", 10)
	logs.Info("config: %v", cfg)

	// create dhcp manager
	// dhcp Select/Release ip for opennotr client
	dhcp, err := core.NewDHCP(cfg.DHCPConfig.Cidr)
	if err != nil {
		logs.Error("new dhcp module fail: %v", err)
		return
	}

	// setup all plugin base on plugin json configuration
	err = plugin.Setup(cfg.Plugins)
	if err != nil {
		logs.Error("setup plugin fail: %v", err)
		return
	}

	// initial resolver
	// currently resolver use coredns and etcd
	// our resolver just write DOMAIN => VIP record to etcd
	var resolver *core.Resolver
	if len(cfg.ResolverConfig.EtcdEndpoints) > 0 {
		resolver, err = core.NewResolve(cfg.ResolverConfig.EtcdEndpoints)
		if err != nil {
			log.Println(err)
			return
		}
	}

	// up local tcp,udp service
	// we use tproxy to route traffic to the tcp port and udp port here.
	go core.NewTCPForward().ListenAndServe(cfg.ServerConfig.TCPForwardListen)
	go core.NewUDPForward().ListenAndServe(cfg.ServerConfig.UDPForwardListen)

	// server provides tcp server for opennotr client
	s := core.NewServer(cfg.ServerConfig, dhcp, resolver)
	fmt.Println(s.ListenAndServe())
}
