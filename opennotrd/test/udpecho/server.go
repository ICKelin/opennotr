package main

import (
	"flag"
	"fmt"
	"net"
)

func main() {
	localAddress := flag.String("l", "", "local address")
	flag.Parse()

	laddr, err := net.ResolveUDPAddr("udp", *localAddress)
	if err != nil {
		fmt.Println(err)
		return
	}

	lconn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer lconn.Close()

	buf := make([]byte, 64*1024)
	for {
		nr, raddr, err := lconn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println(err)
			break
		}
		lconn.WriteToUDP(buf[:nr], raddr)
	}
}
