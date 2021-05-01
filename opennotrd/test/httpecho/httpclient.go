package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func main() {
	remoteAddr := flag.String("r", "", "remote address")
	flag.Parse()

	url := fmt.Sprintf("http://%s/echo", *remoteAddr)
	for i := 0; i < 100; i++ {
		beg := time.Now()
		rsp, err := http.Get(url)
		if err != nil {
			break
		}
		cnt, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			fmt.Println(err)
			break
		}

		rsp.Body.Close()
		fmt.Printf("echo http packet %d %s rtt %dms\n", i+1, string(cnt), time.Now().Sub(beg).Milliseconds())
		time.Sleep(time.Second * 1)
	}
}
