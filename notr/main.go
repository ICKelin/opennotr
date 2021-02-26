package main

import (
	"flag"
	"log"
)

func main() {
	confpath := flag.String("conf", "", "config file path")
	flag.Parse()

	cfg, err := ParseConfig(*confpath)
	if err != nil {
		log.Println(err)
		return
	}

	cli := NewClient(cfg)
	cli.Run()
}
