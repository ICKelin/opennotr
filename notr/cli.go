package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
)

var usage = `./notr [OPTIONS]
Options:
`

var usage1 = `
example:
   ./notr -http 8000 -s 127.0.0.1:9409
   ./notr -https 443 -s 127.0.0.1:9409
   ./notr -tcp 22,25 -s 127.0.0.1:9409
   ./notr -http 8000 -https 443 -tcp 22,25 -s 127.0.0.1:9409
   ./notr -https 443 -auth "YOUR AUTH TOKEN" -s 127.0.0.1:9409
`

type Options struct {
	authKey    string
	serverAddr string
	tcpPorts   []int
	httpPort   int
	httpsPort  int
	supportWin bool
}

func ParseArgs() (*Options, error) {
	flag.Usage = func() {
		fmt.Println(usage)
		flag.PrintDefaults()
		fmt.Println(usage1)
	}

	pserver := flag.String(
		"srv",
		"",
		"server address")

	pkey := flag.String(
		"auth",
		"",
		"authorize token")

	ptcp := flag.String("tcp",
		"",
		"local tcp port list, seperate by \",\"")

	phttp := flag.Int("http",
		0,
		"local http server port")

	phttps := flag.Int("https",
		0,
		"local https server port")

	pversion := flag.Bool("v",
		false,
		"print version")

	flag.Parse()

	if *pversion {
		fmt.Println(version)
		os.Exit(0)
	}

	opts := &Options{
		serverAddr: *pserver,
		httpPort:   *phttp,
		httpsPort:  *phttps,
	}

	if *pkey == "" {
		*pkey = getToken()
	} else {
		saveToken(*pkey)
	}
	opts.authKey = *pkey

	portlist := getPortList(*ptcp)
	opts.tcpPorts = portlist

	opts.supportWin = false
	if runtime.GOOS == "windows" {
		opts.supportWin = true
	}

	return opts, nil
}

func getToken() string {
	b, err := ioutil.ReadFile("./notr.conf")
	if err != nil {
		return ""
	}

	return string(b)
}

func saveToken(token string) {
	fp, err := os.Create("./notr.conf")
	if err != nil {
		fmt.Println("创建授权文件失败：", err)
		return
	}

	defer fp.Close()

	fp.Write([]byte(token))
}

func getPortList(src string) []int {
	portlist := make([]int, 0)
	sp := strings.Split(src, ",")
	for _, p := range sp {
		port, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			continue
		}

		portlist = append(portlist, port)
	}

	return portlist
}
