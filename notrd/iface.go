package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"

	"github.com/songgao/water"
)

type InterfaceConfig struct {
	name   string
	ip     string
	mask   string
	gw     string
	tapDev bool
}

type Interface struct {
	*water.Interface
	name string
	ip   string
	mask string
	gw   string
}

func NewInterface(cfg *InterfaceConfig) (*Interface, error) {
	ifconfig := water.Config{}
	if cfg.tapDev {
		ifconfig.DeviceType = water.TAP
	} else {
		ifconfig.DeviceType = water.TUN
	}

	ifce, err := water.New(ifconfig)
	if err != nil {
		return nil, err
	}

	err = setupDevice(ifce.Name(), cfg.gw)
	if err != nil {
		return nil, err
	}

	iface := &Interface{
		name:      ifce.Name(),
		ip:        cfg.ip,
		gw:        cfg.gw,
		mask:      defaultMask,
		Interface: ifce,
	}
	return iface, nil
}

func setupDevice(dev, ip string) (err error) {
	type CMD struct {
		cmd  string
		args []string
	}

	cmdlist := make([]*CMD, 0)

	cmdlist = append(cmdlist, &CMD{cmd: "ifconfig", args: []string{dev, "up"}})
	switch runtime.GOOS {
	case "linux":
		args := strings.Split(fmt.Sprintf("addr add %s/24 dev %s", ip, dev), " ")
		cmdlist = append(cmdlist, &CMD{cmd: "ip", args: args})

	case "darwin":
		args := strings.Split(fmt.Sprintf("%s %s %s", dev, ip, ip), " ")
		cmdlist = append(cmdlist, &CMD{cmd: "ifconfig", args: args})

		args = strings.Split(fmt.Sprintf("add -net %s/24 %s", ip, ip), " ")
		cmdlist = append(cmdlist, &CMD{cmd: "route", args: args})

	default:
		log.Println("not support windows")
		return
	}

	for _, c := range cmdlist {
		output, err := exec.Command(c.cmd, c.args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("run %s error %s", c, string(output))
		}
	}
	return nil
}
