package server

import "net"

type Session struct {
	conn       net.Conn
	clientAddr string
	activePing int32

	hbbuf    chan struct{}
	writebuf chan []byte
	readbuf  chan []byte
}

func newSession(conn net.Conn, clientAddr string) *Session {
	return &Session{
		conn:       conn,
		clientAddr: clientAddr,
		activePing: 0,
		hbbuf:      make(chan struct{}),
		writebuf:   make(chan []byte),
		readbuf:    make(chan []byte),
	}
}
