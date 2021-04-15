package device

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/songgao/water"
)

type Device struct {
	iface *water.Interface
}

func New() (*Device, error) {
	cfg := water.Config{
		DeviceType: water.TUN,
	}

	iface, err := water.New(cfg)
	if err != nil {
		return nil, err
	}

	return &Device{
		iface: iface,
	}, nil
}

// 设置设备ip地址
func (d *Device) SetIP(gwcidr, vip string) error {
	if runtime.GOOS == "linux" {
		_, err := execCmd("ifconfig", []string{d.iface.Name(), "up"})
		if err != nil {
			return fmt.Errorf("up interface %s %v", d.iface.Name(), err)
		}

		return nil
	}

	if runtime.GOOS == "darwin" {
		_, err := execCmd("ifconfig", []string{d.iface.Name(), "up"})
		if err != nil {
			return fmt.Errorf("up interface %s %v", d.iface.Name(), err)
		}

		_, err = execCmd("ifconfig", []string{d.iface.Name(), vip, vip})
		if err != nil {
			return fmt.Errorf("setup ip address %s fail: %v", vip, err)
		}

		return nil
	}

	return fmt.Errorf("unsupported platform %s", runtime.GOOS)
}

// 设置设备路由
// 针对mac平台，需要设置nexthop_ip/mask和网卡地址
// 针对linux平台，只需要设置ip/mask，会添加默认路由
func (d *Device) SetRoute(nexthop, vip string) error {
	if runtime.GOOS == "linux" {
		sp := strings.Split(nexthop, "/")
		routeAddr := fmt.Sprintf("%s/%s", vip, sp[1])
		_, err := execCmd("ip", []string{"addr", "add", routeAddr, "dev", d.iface.Name()})
		if err != nil {
			return fmt.Errorf("set route %s fail: %v", routeAddr, err)
		}
	}

	if runtime.GOOS == "darwin" {
		args := strings.Split(fmt.Sprintf("add -net %s %s", nexthop, vip), " ")
		_, err := execCmd("route", args)
		if err != nil {
			return fmt.Errorf("set route fail: %v", err)
		}
	}

	return nil
}

func (d *Device) Read() ([]byte, error) {
	buf := make([]byte, 1600)
	nr, err := d.iface.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:nr], nil
}

func (d *Device) Write(buf []byte) (int, error) {
	return d.iface.Write(buf)
}

func (d *Device) Close() {
	d.iface.Close()
}

func execCmd(cmd string, args []string) (string, error) {
	b, err := exec.Command(cmd, args...).CombinedOutput()
	return string(b), err
}
