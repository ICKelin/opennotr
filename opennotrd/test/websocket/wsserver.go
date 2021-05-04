package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/siddontang/go/websocket"
)

func main() {
	localAddress := flag.String("l", "", "local address")
	flag.Parse()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, r.Header)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer conn.Close()

		msgType, msg, err := conn.Read()
		if err != nil {
			fmt.Println(err)
			return
		}
		conn.WriteMessage(msgType, msg)
	})
	http.ListenAndServe(*localAddress, nil)
}
