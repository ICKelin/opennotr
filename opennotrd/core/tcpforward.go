package core

import (
	"io"
	"net"
	"syscall"
	"time"

	"github.com/ICKelin/opennotr/pkg/logs"
)

type TCPForward struct {
	sessMgr *SessionManager
}

func NewTCPForward() *TCPForward {
	return &TCPForward{
		sessMgr: GetSessionManager(),
	}
}

func (f *TCPForward) ListenAndServe(listenAddr string) error {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	// set socket with ip transparent option
	file, err := listener.(*net.TCPListener).File()
	if err != nil {
		return err
	}
	defer file.Close()

	err = syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1)
	if err != nil {
		return err
	}

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
	dip, dport, _ := net.SplitHostPort(conn.LocalAddr().String())
	sip, sport, _ := net.SplitHostPort(conn.RemoteAddr().String())

	sess := f.sessMgr.GetSession(dip)
	if sess == nil {
		logs.Error("no route to host: %s", dip)
		conn.Close()
		return
	}

	stream, err := sess.conn.OpenStream()
	if err != nil {
		logs.Error("open stream fail: %v", err)
		conn.Close()
		return
	}

	bytes := encodeProxyProtocol("tcp", sip, sport, "127.0.0.1", dport)
	stream.SetWriteDeadline(time.Now().Add(time.Second * 10))
	_, err = stream.Write(bytes)
	stream.SetWriteDeadline(time.Time{})
	if err != nil {
		logs.Error("stream write fail: %v", err)
		conn.Close()
		stream.Close()
		return
	}

	go func() {
		defer stream.Close()
		defer conn.Close()
		io.Copy(stream, conn)
	}()

	go func() {
		defer stream.Close()
		defer conn.Close()
		io.Copy(conn, stream)
	}()
}
