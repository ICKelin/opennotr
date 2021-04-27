package stream

import (
	"net"
	"testing"
	"time"
)

func runTCPServer(bufsize int) net.Listener {
	// backend
	lis, err := net.Listen("tcp", "127.0.0.1:2345")
	if err != nil {
		panic(err)
	}

	go func() {
		defer lis.Close()
		for {
			conn, err := lis.Accept()
			if err != nil {
				break
			}

			go onconn(conn, bufsize)
		}
	}()
	return lis
}

func onconn(conn net.Conn, bufsize int) {
	defer conn.Close()
	buf := make([]byte, bufsize)
	nr, _ := conn.Read(buf)
	conn.Write(buf[:nr])
}

func runEcho(t *testing.T, bufsize, numconn int) {
	err := stream.AddProxy("tcp", "127.0.0.1:1234", "127.0.0.1:2345")
	if err != nil {
		t.Error(err)
		return
	}

	lis := runTCPServer(bufsize)

	// client
	for i := 0; i < numconn; i++ {
		go func() {
			buf := make([]byte, bufsize)
			conn, err := net.Dial("tcp", "127.0.0.1:1234")
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()
			for {
				conn.Write(buf)
				conn.Read(buf)
			}
		}()
	}
	tick := time.NewTicker(time.Second * 60)
	<-tick.C

	lis.Close()
}

func TestTCPEcho128B(t *testing.T) {
	numconn := 128
	bufsize := 1024
	runEcho(t, bufsize, numconn)
}

func TestTCPEcho256B(t *testing.T) {
	numconn := 256
	bufsize := 1024
	runEcho(t, bufsize, numconn)
}
func TestTCPEcho512B(t *testing.T) {
	numconn := 512
	bufsize := 1024
	runEcho(t, bufsize, numconn)
}

func TestTCPEcho1K(t *testing.T) {
	numconn := 1024
	bufsize := 1024
	runEcho(t, bufsize, numconn)
}

func TestTCPEcho2K(t *testing.T) {
	numconn := 1024 * 2
	bufsize := 1024
	runEcho(t, bufsize, numconn)
}

func TestUDPPProxy(t *testing.T) {}

func BenchmarkTCPProxy(t *testing.B) {}

func BenchmarkUDPProxy(t *testing.B) {}
