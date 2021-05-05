package core

import (
	"sync"

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
	conn       *yamux.Session
	clientAddr string
	rxbytes    uint64
	txbytes    uint64
}

func newSession(conn *yamux.Session, clientAddr string) *Session {
	return &Session{
		conn:       conn,
		clientAddr: clientAddr,
	}
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
