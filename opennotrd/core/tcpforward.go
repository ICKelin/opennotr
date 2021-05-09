package core

import (
	"io"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/ICKelin/opennotr/pkg/logs"
)

var (
	// default tcp timeout(read, write), 10 seconds
	defaultTCPTimeout = 10
)

type TCPForward struct {
	listenAddr string
	// writeTimeout defines the tcp connection write timeout in second
	// default value set to 10 seconds
	writeTimeout time.Duration

	// readTimeout defines the tcp connection write timeout in second
	// default value set to 10 seconds
	readTimeout time.Duration
	sessMgr     *SessionManager
}

func NewTCPForward(cfg TCPForwardConfig) *TCPForward {
	tcpReadTimeout := cfg.ReadTimeout
	if tcpReadTimeout <= 0 {
		tcpReadTimeout = defaultTCPTimeout
	}

	tcpWriteTimeout := cfg.WriteTimeout
	if tcpWriteTimeout <= 0 {
		tcpWriteTimeout = int(defaultTCPTimeout)
	}
	return &TCPForward{
		listenAddr:   cfg.ListenAddr,
		writeTimeout: time.Duration(tcpWriteTimeout) * time.Second,
		readTimeout:  time.Duration(tcpReadTimeout) * time.Second,
		sessMgr:      GetSessionManager(),
	}
}

func (f *TCPForward) Listen() (net.Listener, error) {
	listener, err := net.Listen("tcp", f.listenAddr)
	if err != nil {
		return nil, err
	}

	// set socket with ip transparent option
	file, err := listener.(*net.TCPListener).File()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	err = syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func (f *TCPForward) Serve(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			logs.Error("accept fail: %v", err)
			break
		}

		go f.forwardTCP(conn)
	}

	return nil
}

func (f *TCPForward) forwardTCP(conn net.Conn) {
	defer conn.Close()

	dip, dport, _ := net.SplitHostPort(conn.LocalAddr().String())
	sip, sport, _ := net.SplitHostPort(conn.RemoteAddr().String())

	sess := f.sessMgr.GetSession(dip)
	if sess == nil {
		logs.Error("no route to host: %s", dip)
		return
	}

	stream, err := sess.conn.OpenStream()
	if err != nil {
		logs.Error("open stream fail: %v", err)
		return
	}
	defer stream.Close()

	// todo rewrite to client configuration
	targetIP := "127.0.0.1"
	bytes := encodeProxyProtocol("tcp", sip, sport, targetIP, dport)
	stream.SetWriteDeadline(time.Now().Add(f.writeTimeout))
	_, err = stream.Write(bytes)
	stream.SetWriteDeadline(time.Time{})
	if err != nil {
		logs.Error("stream write fail: %v", err)
		return
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()

	go func() {
		defer wg.Done()
		defer stream.Close()
		defer conn.Close()
		buf := make([]byte, 4096)
		io.CopyBuffer(stream, conn, buf)
	}()

	// todo: optimize mem alloc
	// one session will cause 4KB + 4KB buffer for io copy
	// and two goroutine 4KB mem used
	buf := make([]byte, 4096)
	io.CopyBuffer(conn, stream, buf)
}
