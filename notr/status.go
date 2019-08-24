package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type Status struct {
	stop chan struct{}
}

func NewStatus() *Status {
	return &Status{
		stop: make(chan struct{}),
	}
}

func (s *Status) Run(notr *Client) {
	for {
		select {
		case <-s.stop:
			return

		default:
		}

		clear()
		fmt.Println("Notr By ICKelin")
		fmt.Printf("%-10s\t %s\n", "状态", "已连接")
		fmt.Printf("%-10s\t %s\n", "域名", notr.domain)
		fmt.Printf("%-10s\t %s\n\n", "客户端ID", notr.authKey)
		fmt.Println("转发列表:")
		fmt.Println("=============================================================")

		idx := 1
		if notr.localHttpsPort != 0 {
			fmt.Printf("%02d. %s => %s:%d\n", idx, "https://"+notr.domain, "127.0.0.1", notr.localHttpsPort)
			idx++
		}

		if notr.localHttpPort != 0 {
			fmt.Printf("%02d. %s => %s:%d\n", idx, "http://"+notr.domain, "127.0.0.1", notr.localHttpPort)
			idx++
		}

		time.Sleep(time.Second * 5)
	}
}

func clear() {
	cmd := exec.Command("clear")
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cls")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func (s *Status) Stop() {
	close(s.stop)
}
