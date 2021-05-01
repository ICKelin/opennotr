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

	buf := make([]byte, 1024)
	for i := 0; i < 100; i++ {
		beg := time.Now()

		conn, err := net.Dial("tcp", *remoteAddr)
		if err != nil {
			fmt.Println(err)
			break
		}

		conn.Write(buf)
		conn.Read(buf)
		conn.Close()
		fmt.Printf("echo tcp packet %d rtt %dms\n", i+1, time.Now().Sub(beg).Milliseconds())
		time.Sleep(time.Second * 1)
	}
}
