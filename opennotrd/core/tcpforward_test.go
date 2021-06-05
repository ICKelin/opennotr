package core

import (
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/xtaci/smux"
)

func init() {
	go http.ListenAndServe("127.0.0.1:6060", nil)
}

// client -----> tproxy | opennotr server <------ opennotr client

var backendAddr = "127.0.0.1:8522"
var serverAddr = "127.0.0.1:8521"
var tproxyAddr = "127.0.0.1:8520"
var vip = "100.64.240.10"

type mockConn struct {
	net.Conn
	addr mockAddr
}

type mockAddr struct{}

func (addr mockAddr) Network() string {
	return "tcp"
}

func (addr mockAddr) String() string {
	return "100.64.240.10:8522"
}

func (c *mockConn) LocalAddr() net.Addr {
	return c.addr
}

func runBackend() {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	sess, err := smux.Client(conn, nil)
	if err != nil {
		panic(err)
	}
	defer sess.Close()

	for {
		stream, err := sess.AcceptStream()
		if err != nil {
			panic(err)
		}

		go func() {
			defer stream.Close()
			buf := make([]byte, len("ping\n"))
			for {
				nr, err := stream.Read(buf)
				if err != nil {
					break
				}
				stream.Write(buf[:nr])
			}
		}()
	}
}

func runserver(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		go func() {
			sess, err := smux.Server(conn, nil)
			if err != nil {
				panic(err)
			}

			sessMgr := GetSessionManager()
			sessMgr.AddSession(vip, &Session{conn: sess})
			fmt.Println("add session: ", vip)
		}()
	}
}

func runtproxy(tcpfw *TCPForward, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		go func() {
			// forward test
			mConn := &mockConn{}
			mConn.Conn = conn
			tcpfw.forwardTCP(mConn)
		}()
	}
}

func TestTCPForward(t *testing.T) {
	// listen tproxy
	tcpfw := NewTCPForward(TCPForwardConfig{
		ListenAddr: tproxyAddr,
	})
	listener, err := tcpfw.Listen()
	if err != nil {
		t.Error(err)
		return
	}
	// defer listener.Close()

	srvlistener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		t.Error(err)
		return
	}
	defer srvlistener.Close()

	go runBackend()
	go runserver(srvlistener)
	go runtproxy(tcpfw, listener)
	// wait for session created
	time.Sleep(time.Second * 1)
	conn, err := net.Dial("tcp", tproxyAddr)
	if err != nil {
		t.FailNow()
	}
	defer conn.Close()

	go func() {
		defer conn.Close()
		for i := 0; i < 10; i++ {
			conn.Write([]byte("ping\n"))
			time.Sleep(time.Second * 1)
		}
		fmt.Println("connection close")
	}()

	buf := make([]byte, 128)
	c := 0
	for {
		nr, err := conn.Read(buf)
		if err != nil {
			break
		}
		fmt.Printf("receive %d %s\n", c+1, string(buf[:nr]))
		c += 1
	}
}

func benchmark(t *testing.B, nconn int) {
	// listen tproxy
	tcpfw := NewTCPForward(TCPForwardConfig{
		ListenAddr: tproxyAddr,
	})
	listener, err := tcpfw.Listen()
	if err != nil {
		t.Error(err)
		return
	}
	// defer listener.Close()

	srvlistener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		t.Error(err)
		return
	}
	defer srvlistener.Close()

	go runBackend()
	go runserver(srvlistener)
	go runtproxy(tcpfw, listener)

	// wait for session created
	time.Sleep(time.Second * 1)
	wg := sync.WaitGroup{}
	wg.Add(nconn)
	defer wg.Wait()
	for i := 0; i < nconn; i++ {
		go func() {
			defer wg.Done()
			conn, err := net.Dial("tcp", tproxyAddr)
			if err != nil {
				t.FailNow()
			}
			defer conn.Close()

			go func() {
				defer conn.Close()
				for i := 0; i < 10; i++ {
					conn.Write([]byte("ping\n"))
					time.Sleep(time.Second * 1)
				}
			}()
			fp, _ := os.Open(os.DevNull)
			defer fp.Close()
			io.Copy(fp, conn)
		}()
	}
}

func Benchmark1K(b *testing.B) {
	benchmark(b, 1024)
}

func Benchmark2K(b *testing.B) {
	benchmark(b, 1024*2)
}

func Benchmark4K(b *testing.B) {
	benchmark(b, 1024*4)
}

func Benchmark8K(b *testing.B) {
	benchmark(b, 1024*8)
}

func Benchmark10K(b *testing.B) {
	benchmark(b, 1024*10)
}

func Benchmark14K(b *testing.B) {
	benchmark(b, 1024*14)
}
