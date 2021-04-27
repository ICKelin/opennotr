package stream

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

func (t *TCPProxy) StopProxy(item *ProxyItem) {
	select {
	case item.RecycleSignal <- struct{}{}:
	default:
	}
}

func (t *TCPProxy) RunProxy(item *ProxyItem) error {
	from, to := item.From, item.To
	lis, err := net.Listen("tcp", from)
	if err != nil {
		return err
	}

	fin := make(chan struct{})
	go func() {
		select {
		case <-item.RecycleSignal:
			logs.Info("receive recycle signal for %s", from)
			lis.Close()
		case <-fin:
			return
		}
	}()

	go func() {
		defer lis.Close()
		defer close(fin)

		sess := &sync.Map{}
		defer func() {
			sess.Range(func(k, v interface{}) bool {
				if conn, ok := v.(net.Conn); ok {
					conn.Close()
				}
				return true
			})
		}()

		for {
			conn, err := lis.Accept()
			if err != nil {
				logs.Error("accept fail: %v", err)
				break
			}

			go func() {
				sess.Store(conn.RemoteAddr().String(), conn)
				defer sess.Delete(conn.RemoteAddr().String())
				t.doProxy(conn, to)
			}()
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
		buf := make([]byte, 1500)
		io.CopyBuffer(toconn, conn, buf)
	}()

	go func() {
		defer wg.Done()
		buf := make([]byte, 1500)
		io.CopyBuffer(conn, toconn, buf)
	}()
	wg.Wait()
}
