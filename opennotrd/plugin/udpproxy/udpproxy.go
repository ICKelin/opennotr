package udpproxy

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/ICKelin/opennotr/internal/logs"
	"github.com/ICKelin/opennotr/opennotrd/plugin"
)

// default timeout for udp session
var defaultTimeout = 30

func init() {
	plugin.Register("udp", &UDPProxy{})
}

type config struct {
	// session timeout(second)
	SessionTimeout int `json:"sessionTimeout"`
}

type UDPProxy struct {
	cfg config
}

func (p *UDPProxy) Setup(rawMessage json.RawMessage) error {
	var cfg config
	err := json.Unmarshal(rawMessage, &cfg)
	if err != nil {
		return err
	}

	if cfg.SessionTimeout <= 0 {
		cfg.SessionTimeout = defaultTimeout
	}
	p.cfg = cfg
	return nil
}

func (p *UDPProxy) StopProxy(item *plugin.PluginMeta) {
	select {
	case item.RecycleSignal <- struct{}{}:
	default:
	}
}

func (p *UDPProxy) RunProxy(item *plugin.PluginMeta) (*plugin.ProxyTuple, error) {
	from := item.From
	laddr, err := net.ResolveUDPAddr("udp", from)
	if err != nil {
		return nil, err
	}

	lis, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, err
	}

	go p.doProxy(lis, item)

	_, fromPort, _ := net.SplitHostPort(lis.LocalAddr().String())
	_, toPort, _ := net.SplitHostPort(item.To)

	return &plugin.ProxyTuple{
		Protocol: item.Protocol,
		FromPort: fromPort,
		ToPort:   toPort,
	}, nil
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
	sess := sync.Map{}

	// sessionTimeout store all session key active time
	// the purpose of this is to avoid too session without expired
	sessionTimeout := sync.Map{}

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

	go func() {
		timeout := p.cfg.SessionTimeout
		interval := timeout / 2
		if interval <= 0 {
			interval = timeout
		}
		tick := time.NewTicker(time.Second * time.Duration(interval))
		for range tick.C {
			sessionTimeout.Range(func(k, v interface{}) bool {
				lastActiveAt, ok := v.(time.Time)
				if !ok {
					return true
				}

				if time.Now().Sub(lastActiveAt).Seconds() > float64(timeout) {
					sess.Delete(k)
				}
				return true
			})
		}
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
			sessionTimeout.Store(key, time.Now())

			// read from $to address and write to $from address
			go p.udpCopy(lis, backendConn, raddr)
		}

		val, ok = sess.Load(key)
		if !ok {
			continue
		}

		sessionTimeout.Store(key, time.Now())
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
