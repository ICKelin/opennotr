package core

import (
	"sync"
	"sync/atomic"

	"github.com/hashicorp/yamux"
)

var sessionMgr = &SessionManager{}

type SessionManager struct {
	sessions sync.Map
}

func GetSessionManager() *SessionManager {
	return sessionMgr
}

type Session struct {
	conn    *yamux.Session
	rxbytes uint64
	txbytes uint64
}

func newSession(conn *yamux.Session, vip string) *Session {
	return &Session{
		conn: conn,
	}
}

func (s *Session) ResetRx() uint64 {
	return atomic.SwapUint64(&s.rxbytes, 0)
}

func (s *Session) ResetTx() uint64 {
	return atomic.SwapUint64(&s.txbytes, 0)
}

func (s *Session) IncRx(nb uint64) {
	atomic.AddUint64(&s.rxbytes, nb)
}

func (s *Session) IncTx(nb uint64) {
	atomic.AddUint64(&s.txbytes, nb)
}

func (mgr *SessionManager) AddSession(vip string, sess *Session) {
	mgr.sessions.Store(vip, sess)
}

func (mgr *SessionManager) GetSession(vip string) *Session {
	val, ok := mgr.sessions.Load(vip)
	if !ok {
		return nil
	}
	return val.(*Session)
}

func (mgr *SessionManager) DeleteSession(vip string) {
	mgr.sessions.Delete(vip)
}
