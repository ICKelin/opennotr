package main

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/ICKelin/opennotr/pkg/logs"
)

func init() {
	RegisterProxier("tcp", &TCPProxy{})
}

type TCPProxy struct{}

func (t *TCPProxy) RunProxy(item *ProxyItem) error {
	from, to := item.From, item.To
	lis, err := net.Listen("tcp", from)
	if err != nil {
		return err
	}

	fin := make(chan struct{})
	go func() {
		select {
		case <-item.recycleSignal:
			logs.Info("receive recycle signal for %s", from)
			lis.Close()
		case <-fin:
			return
		}
	}()

	go func() {
		defer lis.Close()
		defer close(fin)

		for {
			conn, err := lis.Accept()
			if err != nil {
				logs.Error("accept fail: %v", err)
				break
			}

			go t.doProxy(conn, to)
		}
	}()

	return nil
}

func (t *TCPProxy) doProxy(conn net.Conn, to string) {
	defer conn.Close()

	toconn, err := net.DialTimeout("tcp", to, time.Second*10)
	if err != nil {
		logs.Error("dial fail: %v", err)
		return
	}
	defer toconn.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(toconn, conn)
	}()

	go func() {
		defer wg.Done()
		io.Copy(conn, toconn)
	}()
	wg.Wait()
}
