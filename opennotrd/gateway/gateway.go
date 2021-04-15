package gateway

import (
	"fmt"
	"net"
	"sync"
)

type Gateway struct {
	rw    sync.RWMutex
	cidr  string
	free  map[string]struct{}
	inuse map[string]struct{}
}

func New(cidr string) *Gateway {
	begin, end, err := parseCIDR(cidr)
	if err != nil {
		// 初始化未成功，panic down掉进程
		panic(err)
	}

	free := make(map[string]struct{})
	for i := begin; i < end; i++ {
		free[toIP(i)] = struct{}{}
	}

	return &Gateway{
		free:  free,
		inuse: make(map[string]struct{}),
		cidr:  cidr,
	}
}

func (g *Gateway) GetAddr() string {
	return g.cidr
}

// 为客户端选择ip
func (g *Gateway) SelectIP() (string, error) {
	g.rw.Lock()
	defer g.rw.Unlock()

	for ip, _ := range g.free {
		delete(g.free, ip)
		g.inuse[ip] = struct{}{}
		return ip, nil
	}

	return "", fmt.Errorf("no available ip")
}

// 释放客户端ip
func (g *Gateway) ReleaseIP(ip string) {
	g.rw.Lock()
	defer g.rw.Unlock()

	g.free[ip] = struct{}{}
	delete(g.inuse, ip)
}

// 解析CIDR
// 解析成功，返回开始地址，结束地址
func parseCIDR(cidr string) (int32, int32, error) {
	ip, mask, err := net.ParseCIDR(cidr)
	if err != nil {
		return -1, -1, err
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return -1, -1, fmt.Errorf("parse cidr fail")
	}

	one, _ := mask.Mask.Size()
	begin := (int32(ipv4[0]) << 24) + (int32(ipv4[1]) << 16) + (int32(ipv4[2]) << 8) + int32(ipv4[3])
	end := begin | (1<<(32-one) - 1)

	return begin + 1, end, nil
}

// int32 ip地址转换为string
func toIP(iip int32) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(iip>>24), byte(iip>>16), byte(iip>>8), byte(iip))
}
