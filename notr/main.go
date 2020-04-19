package main

import (
	"flag"
	"log"

	"github.com/ICKelin/opennotr/notr/client"
	"github.com/ICKelin/opennotr/notr/config"
)

func main() {
	confpath := flag.String("conf", "", "config file path")
	flag.Parse()

	cfg, err := config.Parse(*confpath)
	if err != nil {
		log.Println(err)
		return
	}

	cli := client.New(cfg)
	cli.Run()
}
