package main

import (
	"flag"
	"fmt"
	"net"
)

func main() {
	localAddress := flag.String("l", "", "local address")
	flag.Parse()

	listener, err := net.Listen("tcp", *localAddress)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		go func() {
			defer conn.Close()
			buf := make([]byte, 1024)
			nr, err := conn.Read(buf)
			if err != nil {
				fmt.Println(err)
				return
			}

			conn.Write(buf[:nr])
		}()
	}
}
