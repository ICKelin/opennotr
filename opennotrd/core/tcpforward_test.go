package core

import (
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/yamux"
)

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

func (c *mockConn) Write(buf []byte) (int, error) {
	fmt.Printf("receive %d bytes\n", len(buf))
	return len(buf), nil
}

func runBackend(t *testing.T) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()
	sess, err := yamux.Client(conn, nil)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer sess.Close()

	for {
		stream, err := sess.AcceptStream()
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		go func() {
			defer stream.Close()
			buf := make([]byte, len("ping\n"))
			for {
				nr, err := stream.Read(buf)
				if err != nil {
					fmt.Println(err)
					break
				}
				stream.Write(buf[:nr])
			}
		}()
	}
}

func runserver(t *testing.T, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		go func() {
			sess, err := yamux.Server(conn, nil)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			sessMgr := GetSessionManager()
			sessMgr.AddSession(vip, &Session{conn: sess})
			t.Log("add session: ", vip)
		}()
	}
}

func runtproxy(t *testing.T, tcpfw *TCPForward, listener net.Listener) {
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
	tcpfw := NewTCPForward()
	listener, err := tcpfw.Listen(tproxyAddr)
	if err != nil {
		t.Error(err)
		return
	}
	defer listener.Close()

	srvlistener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		t.Error(err)
		return
	}
	defer srvlistener.Close()

	go runBackend(t)
	go runserver(t, srvlistener)
	go runtproxy(t, tcpfw, listener)

	conn, err := net.Dial("tcp", tproxyAddr)
	if err != nil {
		t.FailNow()
	}
	defer conn.Close()

	go func() {
		for i := 0; i < 100; i++ {
			conn.Write([]byte("ping\n"))
			time.Sleep(time.Second * 1)
		}
	}()

	io.Copy(os.Stdout, conn)
}
