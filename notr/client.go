package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ICKelin/opennotr/common"
	"github.com/google/tcpproxy"
	"github.com/songgao/water"
)

type ClientConfig struct {
	authKey        string
	serverAddr     string
	ctrlKey        string
	isWin          bool
	localHttpPort  int
	localHttpsPort int
	tcpports       []int
}

type Client struct {
	// 预设配置
	authKey        string
	tcpports       []int
	localHttpPort  int
	localHttpsPort int
	serverAddr     string
	stop           chan struct{}

	// 初始化生成配置
	proxyHttpPort  int              // http代理端口
	proxyHttpsPort int              // https代理端口
	iface          *water.Interface // 虚拟网卡句柄

	// 服务器返回配置
	myip        string   // 分配的ip地址
	gateway     string   // 网关
	domain      string   // 分配的域名
	cnames      []string // 别名
	httpsdomain string   // 分配的域名
}

func NewClient(cfg *ClientConfig) (*Client, error) {
	client := &Client{
		serverAddr:     cfg.serverAddr,
		authKey:        cfg.authKey,
		localHttpPort:  cfg.localHttpPort,
		localHttpsPort: cfg.localHttpsPort,
		tcpports:       cfg.tcpports,
		stop:           make(chan struct{}),
	}

	// init http proxy
	if cfg.localHttpPort != 0 {
		port := randPort(2000, 3000)
		if port == -1 {
			return nil, fmt.Errorf("暂无可用网络端口")
		}
		client.proxyHttpPort = port
		go proxy2LocalHost(fmt.Sprintf(":%d", port), fmt.Sprintf("127.0.0.1:%d", cfg.localHttpPort))
	}

	// init https porxy
	if cfg.localHttpsPort != 0 {
		port := randPort(3000, 4000)
		if port == -1 {
			return nil, fmt.Errorf("暂无可用网络端口")
		}
		client.proxyHttpsPort = port
		go proxy2LocalHost(fmt.Sprintf(":%d", port), fmt.Sprintf("127.0.0.1:%d", cfg.localHttpsPort))
	}
	return client, nil
}

func (client *Client) Run() error {
	sndqueue := make(chan []byte)
	for {
		conn, err := net.DialTimeout("tcp", client.serverAddr, time.Second*10)
		if err != nil {
			fmt.Println("与超级节点建立连接失败:", err)
			time.Sleep(time.Second * 3)
			continue
		}

		s2c, err := authorize(client, conn)
		if err != nil {
			conn.Close()
			fmt.Println(err)
			time.Sleep(time.Second * 3)
			continue
		}

		client.myip = s2c.AccessIP
		client.gateway = s2c.Gateway
		client.domain = s2c.Domain

		iface, err := NewIfce()
		if err != nil {
			fmt.Println("notr启动失败，请确认是否使用root启动: ", err)
			return err
		}

		go readIface(iface, sndqueue)

		err = setupDevice(iface.Name(), client.myip, client.gateway)
		if err != nil {
			conn.Close()
			fmt.Println("notr配置失败:", err)
			time.Sleep(time.Second * 3)
			continue
		}

		s := NewStatus()
		go s.Run(client)

		wg := &sync.WaitGroup{}
		wg.Add(3)

		stopChan := make(chan struct{})
		go heartbeat(sndqueue, wg, stopChan)
		go snd(conn, sndqueue, wg, stopChan)
		go rcv(conn, iface, wg)
		wg.Wait()
		conn.Close()

		releaseDevice(iface.Name(), client.myip, client.gateway)
		iface.Close()

		s.Stop()
		fmt.Println("正在重连...")
	}
}

func proxy2LocalHost(fromAddr, toAddr string) {
	var p tcpproxy.Proxy
	p.AddRoute(fromAddr, tcpproxy.To(toAddr))
	p.Run()
}

func readIface(ifce *water.Interface, sndqueue chan []byte) {
	packet := make([]byte, 65536)
	for {
		n, err := ifce.Read(packet)
		if err != nil {
			fmt.Println(err)
			break
		}

		bytes := common.Encode(common.C2C_DATA, packet[:n])
		sndqueue <- bytes
	}
}

func authorize(client *Client, conn net.Conn) (s2cauthorize *common.S2CAuthorize, err error) {
	c2sauthorize := &common.C2SAuthorize{
		Key:       client.authKey,
		HttpPort:  client.proxyHttpPort,
		HttpsPort: client.proxyHttpsPort,
	}

	payload, err := json.Marshal(c2sauthorize)
	if err != nil {
		return nil, err
	}

	buff := common.Encode(common.C2S_AUTHORIZE, payload)

	conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
	_, err = conn.Write(buff)
	conn.SetWriteDeadline(time.Time{})
	if err != nil {
		return nil, err
	}

	cmd, resp, err := common.Decode(conn)
	if err != nil {
		return nil, err
	}

	if cmd != common.S2C_AUTHORIZE {
		err = fmt.Errorf("invalid authorize cmd %d", cmd)
		return nil, err
	}

	var s2c = common.S2CAuthorize{}
	err = common.BodyObj(resp, &s2c)
	return &s2c, err
}

func setupDevice(device, ip, gateway string) (err error) {
	type CMD struct {
		cmd  string
		args []string
	}

	cmdlist := make([]*CMD, 0)

	switch runtime.GOOS {
	case "linux":
		cmdlist = append(cmdlist, &CMD{cmd: "ifconfig", args: []string{device, "up"}})
		args := strings.Split(fmt.Sprintf("addr add %s/24 dev %s", ip, device), " ")
		cmdlist = append(cmdlist, &CMD{cmd: "ip", args: args})

	case "darwin":
		cmdlist = append(cmdlist, &CMD{cmd: "ifconfig", args: []string{device, "up"}})

		args := strings.Split(fmt.Sprintf("%s %s %s", device, ip, ip), " ")
		cmdlist = append(cmdlist, &CMD{cmd: "ifconfig", args: args})

		args = strings.Split(fmt.Sprintf("add -net %s/24 %s", gateway, ip), " ")
		cmdlist = append(cmdlist, &CMD{cmd: "route", args: args})

	default:
		s := fmt.Sprintf("interface ip set address name=\"%s\" addr=%s source=static mask=255.255.255.0 gateway=%s", device, ip, gateway)
		args := strings.Split(s, " ")
		cmdlist = append(cmdlist, &CMD{cmd: "netsh", args: args})

		s = fmt.Sprintf("delete 0.0.0.0 %s", gateway)
		args = strings.Split(s, " ")
		cmdlist = append(cmdlist, &CMD{cmd: "route", args: args})

		// netsh advfirewall firewall add rule name="Allow Notr" dir=in action=allow program="d:\notr.exe"
		wd, _ := filepath.Abs(os.Args[0])
		s = fmt.Sprintf("advfirewall firewall add rule name=\"%s\" dir=in action=allow program=\"%s\"", os.Args[0], wd)
		args = strings.Split(s, " ")
		cmdlist = append(cmdlist, &CMD{cmd: "netsh", args: args})

		// netsh advfirewall firewall delete rule name="notr.exe"
		s = fmt.Sprintf("advfirewall firewall delete rule name=\"%s\"", os.Args[0])
		args = strings.Split(s, " ")
		cmdlist = append(cmdlist, &CMD{cmd: "netsh", args: args})

		s = fmt.Sprintf("advfirewall firewall add rule name=\"%s\" dir=in action=allow program=\"%s\"", os.Args[0], wd)
		args = strings.Split(s, " ")
		cmdlist = append(cmdlist, &CMD{cmd: "netsh", args: args})
	}

	for _, c := range cmdlist {
		output, _ := exec.Command(c.cmd, c.args...).CombinedOutput()
		if err != nil {
			fmt.Printf("run %s error %s\n", c, string(output))
		}
	}

	return nil
}

func releaseDevice(device, ip, gateway string) (err error) {
	type CMD struct {
		cmd  string
		args []string
	}

	cmdlist := make([]*CMD, 0)

	switch runtime.GOOS {
	case "linux":
		args := strings.Split(fmt.Sprintf("%s down", device), " ")
		cmdlist = append(cmdlist, &CMD{cmd: "ifconfig", args: args})

	case "darwin":
		gw := strings.Split(gateway, ".")
		if len(gw) != 4 {
			break
		}

		s := strings.Join(gw[:3], ".")
		args := strings.Split(fmt.Sprintf("delete -net %s/24 %s", s, ip), " ")
		cmdlist = append(cmdlist, &CMD{cmd: "route", args: args})
		args = strings.Split(fmt.Sprintf("%s delete %s", device, ip), " ")
		cmdlist = append(cmdlist, &CMD{cmd: "ifconfig", args: args})
	}

	for _, c := range cmdlist {
		output, _ := exec.Command(c.cmd, c.args...).CombinedOutput()
		if err != nil {
			fmt.Printf("run %s error %s\n", c, string(output))
		}
	}

	return nil
}

func heartbeat(sndqueue chan []byte, wg *sync.WaitGroup, stopChan chan struct{}) {
	defer wg.Done()

	for {
		select {
		case <-stopChan:
			return

		case <-time.After(time.Second * 3):
			bytes := common.Encode(common.C2S_HEARTBEAT, nil)
			sndqueue <- bytes
		}
	}
}

func rcv(conn net.Conn, ifce *water.Interface, wg *sync.WaitGroup) {
	defer conn.Close()
	defer wg.Done()

	for {
		cmd, pkt, err := common.Decode(conn)
		if err != nil {
			fmt.Println("与服务器断开连接: ", err)
			break
		}
		switch cmd {
		case common.S2C_HEARTBEAT:

		case common.C2C_DATA:
			_, err := ifce.Write(pkt)
			if err != nil {
				fmt.Println(err)
			}

		}
	}
}

func snd(conn net.Conn, sndqueue chan []byte, wg *sync.WaitGroup, stopChan chan struct{}) {
	defer conn.Close()
	defer wg.Done()
	defer close(stopChan)

	for {
		pkt := <-sndqueue
		conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
		_, err := conn.Write(pkt)
		conn.SetWriteDeadline(time.Time{})
		if err != nil {
			// fmt.Println(err)
			break
		}
	}
}

func randPort(begin, end int) int {
	for i := begin; i < end; i++ {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", i))
		if err != nil {
			continue
		}
		lis.Close()
		return i
	}
	return -1
}
