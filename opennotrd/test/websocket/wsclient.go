package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	raddr := flag.String("r", "", "remote address")
	flag.Parse()
	buf := make([]byte, 1024)
	for i := 0; i < 100; i++ {
		beg := time.Now()
		conn, _, err := websocket.DefaultDialer.Dial(*raddr, nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		defer conn.Close()
		conn.WriteMessage(websocket.BinaryMessage, buf)
		conn.ReadMessage()
		fmt.Printf("echo websocket packet %d rtt %dms\n", i+1, time.Now().Sub(beg).Milliseconds())
		time.Sleep(time.Second * 1)
	}
}
