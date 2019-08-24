package main

import (
	"fmt"
)

var (
	version string
)

func main() {
	opts, err := ParseArgs()
	if err != nil {
		fmt.Println(err)
		return
	}

	clientCfg := &ClientConfig{
		authKey:        opts.authKey,
		serverAddr:     opts.serverAddr,
		isWin:          opts.supportWin,
		localHttpPort:  opts.httpPort,
		localHttpsPort: opts.httpsPort,
		tcpports:       opts.tcpPorts,
	}

	client, err := NewClient(clientCfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	client.Run()
}
