package main

import (
	"flag"
	"fmt"
	"net"
	"time"
)

func main() {
	remoteAddr := flag.String("r", "", "remote address")
	flag.Parse()

	rconn, err := net.Dial("udp", *remoteAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	buf := make([]byte, 1024)
	for i := 0; i < 100; i++ {
		beg := time.Now()
		_, err := rconn.Write(buf)
		if err != nil {
			fmt.Println(err)
			break
		}

		rconn.Read(buf)
		fmt.Printf("echo udp packet %d rtt %dms\n", i+1, time.Now().Sub(beg).Milliseconds())
		time.Sleep(time.Second * 1)
	}
}
