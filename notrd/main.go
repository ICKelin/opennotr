package main

import (
	"flag"
	"log"
)

var (
	flgConf = flag.String("conf", "./etc/notrd.conf", "config file path")
	version string
)

func main() {
	flag.Parse()
	conf, err := ParseConfig(*flgConf)
	if err != nil {
		log.Println("parse config fail: ", err)
		return
	}

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	s, err := NewServer(&ServerConfig{
		serverAddr: conf.LocalListener,
		deviceIP:   conf.LocalIP,
		clients:    conf.Clients,
		tapDev:     conf.Tap,
	})

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(s.Serve())
}
