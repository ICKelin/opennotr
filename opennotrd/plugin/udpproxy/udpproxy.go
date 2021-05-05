package udpproxy

import (
	"context"
	"encoding/json"
	"net"
	"sync"

	"github.com/ICKelin/opennotr/opennotrd/plugin"
	"github.com/ICKelin/opennotr/pkg/logs"
)

func init() {
	plugin.Register("udp", &UDPProxy{})
}

type UDPProxy struct{}

func (p *UDPProxy) Setup(config json.RawMessage) error { return nil }

func (p *UDPProxy) StopProxy(item *plugin.PluginMeta) {
	select {
	case item.RecycleSignal <- struct{}{}:
	default:
	}
}

func (p *UDPProxy) RunProxy(item *plugin.PluginMeta) error {
	from := item.From
	laddr, err := net.ResolveUDPAddr("udp", from)
	if err != nil {
		return err
	}

	lis, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}

	go p.doProxy(lis, item)
	return nil
}

func (p *UDPProxy) doProxy(lis *net.UDPConn, item *plugin.PluginMeta) {
	defer lis.Close()

	from := item.From
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// receive this signal and close the listener
	// the listener close will force lis.ReadFromUDP loop break
	// then close all the client socket and end udpCopy
	go func() {
		select {
		case <-item.RecycleSignal:
			logs.Info("receive recycle signal for %s", from)
			lis.Close()
		case <-ctx.Done():
			return
		}
	}()

	// sess store all backend connection
	// key: client address
	// value: *net.UDPConn
	// todo: optimize session timeout
	sess := sync.Map{}

	// close all backend sockets
	// this action may end udpCopy
	defer func() {
		sess.Range(func(k, v interface{}) bool {
			if conn, ok := v.(*net.UDPConn); ok {
				conn.Close()
			}
			return true
		})
	}()

	var buf = make([]byte, 64*1024)
	for {
		nr, raddr, err := lis.ReadFromUDP(buf)
		if err != nil {
			logs.Error("read from udp fail: %v", err)
			break
		}

		key := raddr.String()
		val, ok := sess.Load(key)

		if !ok {
			backendAddr, err := net.ResolveUDPAddr("udp", item.To)
			if err != nil {
				logs.Error("resolve udp fail: %v", err)
				break
			}

			backendConn, err := net.DialUDP("udp", nil, backendAddr)
			if err != nil {
				logs.Error("dial udp fail: %v", err)
				break
			}
			sess.Store(key, backendConn)

			// read from $to address and write to $from address
			go p.udpCopy(lis, backendConn, raddr)
		}

		val, _ = sess.Load(key)
		// read from $from address and write to $to address
		val.(*net.UDPConn).Write(buf[:nr])
	}
}

func (p *UDPProxy) udpCopy(dst, src *net.UDPConn, toaddr *net.UDPAddr) {
	defer src.Close()
	buf := make([]byte, 64*1024)
	for {
		nr, _, err := src.ReadFromUDP(buf)
		if err != nil {
			logs.Error("read from udp fail: %v", err)
			break
		}

		_, err = dst.WriteToUDP(buf[:nr], toaddr)
		if err != nil {
			logs.Error("write to udp fail: %v", err)
			break
		}
	}
}
