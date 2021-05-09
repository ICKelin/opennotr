package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/ICKelin/opennotr/pkg/logs"
	"github.com/hashicorp/yamux"
)

var (
	// default udp timeout(read, write)(seconds)
	defaultUDPTimeout = 10

	// default udp session timeout(seconds)
	defaultUDPSessionTimeout = 30
)

// udpSession defines each client forward stream
// the purpose of udpSession is to reuse stream
// tha lastActive members will reset for easch packet in/out
type udpSession struct {
	stream     *yamux.Stream
	lastActive time.Time
}

type udpSessionManager struct {
	sessLock    sync.Mutex
	udpsessions map[string]*udpSession
}

func newUDPSessionManager() *udpSessionManager {
	return &udpSessionManager{
		udpsessions: make(map[string]*udpSession),
	}
}

func (mgr *udpSessionManager) Add(key string, val *udpSession) {
	mgr.sessLock.Lock()
	defer mgr.sessLock.Unlock()
	mgr.udpsessions[key] = val
}

func (mgr *udpSessionManager) Get(key string) *udpSession {
	mgr.sessLock.Lock()
	defer mgr.sessLock.Unlock()
	return mgr.udpsessions[key]
}

func (mgr *udpSessionManager) Delete(key string) {
	mgr.sessLock.Lock()
	defer mgr.sessLock.Unlock()
	delete(mgr.udpsessions, key)
}

func (mgr *udpSessionManager) ResetActive(key string, tm time.Time) {
	mgr.sessLock.Lock()
	defer mgr.sessLock.Unlock()
	sess, ok := mgr.udpsessions[key]
	if !ok {
		return
	}
	sess.lastActive = tm
}

type UDPForward struct {
	listenAddr     string
	sessionTimeout int
	readTimeout    time.Duration
	writeTimeout   time.Duration
	sessMgr        *SessionManager
	udpSessionMgr  *udpSessionManager
}

func NewUDPForward(cfg UDPForwardConfig) *UDPForward {
	readTimeout := cfg.ReadTimeout
	if readTimeout <= 0 {
		readTimeout = defaultUDPTimeout
	}

	writeTimeout := cfg.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = defaultUDPTimeout
	}

	sessionTimeout := cfg.SessionTimeout
	if sessionTimeout <= 0 {
		sessionTimeout = defaultUDPSessionTimeout
	}

	return &UDPForward{
		listenAddr:     cfg.ListenAddr,
		readTimeout:    time.Duration(readTimeout) * time.Second,
		writeTimeout:   time.Duration(writeTimeout) * time.Second,
		sessionTimeout: sessionTimeout,
		sessMgr:        GetSessionManager(),
		udpSessionMgr:  newUDPSessionManager(),
	}
}

func (f *UDPForward) ListenAndServe() error {
	laddr, err := net.ResolveUDPAddr("udp", f.listenAddr)
	if err != nil {
		logs.Error("resolve udp fail: %v", err)
		return err
	}

	lconn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}

	// set socket with ip transparent option
	file, err := lconn.File()
	if err != nil {
		return err
	}
	defer file.Close()

	err = syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1)
	if err != nil {
		return err
	}

	// set socket with recv origin dst option
	err = syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_IP, syscall.IP_RECVORIGDSTADDR, 1)
	if err != nil {
		return err
	}

	// create raw socket fd
	// we use rawsocket to send udp packet back to client.
	rawfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil || rawfd < 0 {
		logs.Error("call socket fail: %v", err)
		return err
	}
	defer syscall.Close(rawfd)

	err = syscall.SetsockoptInt(rawfd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1)
	if err != nil {
		return err
	}

	go f.recyeleSession()
	buf := make([]byte, 64*1024)
	oob := make([]byte, 1024)
	for {
		// udp is not connect oriented, it should use read message
		// and read the origin dst ip and port from msghdr
		nr, oobn, _, raddr, err := lconn.ReadMsgUDP(buf, oob)
		if err != nil {
			logs.Error("read from udp fail: %v", err)
			break
		}

		origindst, err := f.getOriginDst(oob[:oobn])
		if err != nil {
			logs.Error("get origin dst fail: %v", err)
			continue
		}

		dip, dport, _ := net.SplitHostPort(origindst.String())
		sip, sport, _ := net.SplitHostPort(raddr.String())

		key := fmt.Sprintf("%s:%s:%s:%s", sip, sport, dip, dport)
		udpsess := f.udpSessionMgr.Get(key)
		if udpsess != nil {
			sess := f.sessMgr.GetSession(dip)
			if sess == nil {
				logs.Error("no route to host: %s", dip)
				continue
			}

			stream, err := sess.conn.OpenStream()
			if err != nil {
				logs.Error("open stream fail: %v", err)
				continue
			}

			udpsess = &udpSession{stream, time.Now()}
			f.udpSessionMgr.Add(key, udpsess)
			targetIP := "127.0.0.1"
			bytes := encodeProxyProtocol("udp", sip, sport, targetIP, dport)
			stream.SetWriteDeadline(time.Now().Add(f.writeTimeout))
			_, err = stream.Write(bytes)
			stream.SetWriteDeadline(time.Time{})
			if err != nil {
				logs.Error("stream write fail: %v", err)
				continue
			}

			go f.forwardUDP(stream, key, rawfd, origindst, raddr)
		}

		f.udpSessionMgr.ResetActive(key, time.Now())
		stream := udpsess.stream

		bytes := encode(buf[:nr])
		stream.SetWriteDeadline(time.Now().Add(f.writeTimeout))
		_, err = stream.Write(bytes)
		stream.SetWriteDeadline(time.Time{})
		if err != nil {
			logs.Error("stream write fail: %v", err)
		}
	}
	return nil
}

// forwardUDP reads from stream and write to tofd via rawsocket
func (f *UDPForward) forwardUDP(stream *yamux.Stream, sessionKey string, tofd int, fromaddr, toaddr *net.UDPAddr) {
	defer stream.Close()
	defer f.udpSessionMgr.Delete(sessionKey)
	hdr := make([]byte, 2)
	for {
		nr, err := stream.Read(hdr)
		if err != nil {
			if err != io.EOF {
				logs.Error("read stream fail %v", err)
			}
			break
		}
		if nr != 2 {
			logs.Error("invalid bodylen: %d", nr)
			continue
		}

		nlen := binary.BigEndian.Uint16(hdr)
		buf := make([]byte, nlen)
		stream.SetReadDeadline(time.Now().Add(f.readTimeout))
		_, err = io.ReadFull(stream, buf)
		stream.SetReadDeadline(time.Time{})
		if err != nil {
			logs.Error("read stream body fail: %v", err)
			break
		}

		err = sendUDPViaRaw(tofd, fromaddr, toaddr, buf)
		if err != nil {
			logs.Error("send via raw socket fail: %v", err)
		}
		f.udpSessionMgr.ResetActive(sessionKey, time.Now())
	}
}

func (f *UDPForward) recyeleSession() {
	tick := time.NewTicker(time.Second * 5)
	for range tick.C {
		total, timeout := len(f.udpSessionMgr.udpsessions), 0
		f.udpSessionMgr.sessLock.Lock()
		for k, s := range f.udpSessionMgr.udpsessions {
			if time.Now().Sub(s.lastActive).Seconds() > float64(f.sessionTimeout) {
				logs.Warn("remove udp %v session, lastActive: %v", k, s.lastActive)
				timeout += 1
				delete(f.udpSessionMgr.udpsessions, k)
			}
		}
		f.udpSessionMgr.sessLock.Unlock()
		logs.Debug("total %d, timeout %d, left: %d", total, timeout, total-timeout)
	}
}

func (f *UDPForward) getOriginDst(hdr []byte) (*net.UDPAddr, error) {
	msgs, err := syscall.ParseSocketControlMessage(hdr)
	if err != nil {
		return nil, err
	}

	var origindst *net.UDPAddr
	for _, msg := range msgs {
		if msg.Header.Level == syscall.SOL_IP &&
			msg.Header.Type == syscall.IP_RECVORIGDSTADDR {
			originDstRaw := &syscall.RawSockaddrInet4{}
			err := binary.Read(bytes.NewReader(msg.Data), binary.LittleEndian, originDstRaw)
			if err != nil {
				logs.Error("read origin dst fail: %v", err)
				continue
			}

			// only support for ipv4
			if originDstRaw.Family == syscall.AF_INET {
				pp := (*syscall.RawSockaddrInet4)(unsafe.Pointer(originDstRaw))
				p := (*[2]byte)(unsafe.Pointer(&pp.Port))
				origindst = &net.UDPAddr{
					IP:   net.IPv4(pp.Addr[0], pp.Addr[1], pp.Addr[2], pp.Addr[3]),
					Port: int(p[0])<<8 + int(p[1]),
				}
			}
		}
	}

	if origindst == nil {
		return nil, fmt.Errorf("get origin dst fail")
	}

	return origindst, nil
}
