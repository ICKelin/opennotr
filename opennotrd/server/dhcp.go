package server

import (
	"fmt"
	"net"
	"sync"
)

type DHCP struct {
	rw      sync.Mutex
	cidr    string
	localIP string
	free    map[string]struct{}
	inuse   map[string]struct{}
}

func NewDHCP(cidr string) (*DHCP, error) {
	begin, end, err := getIPRange(cidr)
	if err != nil {
		return nil, err
	}

	free := make(map[string]struct{})
	inused := make(map[string]struct{})
	for i := begin + 1; i < end; i++ {
		free[toIP(i)] = struct{}{}
	}

	return &DHCP{
		free:    free,
		inuse:   inused,
		cidr:    cidr,
		localIP: toIP(begin),
	}, nil
}

func (g *DHCP) GetCIDR() string {
	return g.cidr
}

func (g *DHCP) SelectIP() (string, error) {
	g.rw.Lock()
	defer g.rw.Unlock()

	for ip := range g.free {
		delete(g.free, ip)
		g.inuse[ip] = struct{}{}
		return ip, nil
	}

	return "", fmt.Errorf("no available ip")
}

func (g *DHCP) ReleaseIP(ip string) {
	g.rw.Lock()
	defer g.rw.Unlock()

	g.free[ip] = struct{}{}
	delete(g.inuse, ip)
}

func getIPRange(cidr string) (int32, int32, error) {
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

	return begin, end, nil
}

// int32 ip地址转换为string
func toIP(iip int32) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(iip>>24), byte(iip>>16), byte(iip>>8), byte(iip))
}
