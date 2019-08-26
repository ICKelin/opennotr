package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/ICKelin/opennotr/common"
)

type ForwardConfig struct {
}
type Forward struct {
	table *sync.Map
}

func NewForward() *Forward {
	forwardTable := &Forward{
		table: new(sync.Map),
	}
	return forwardTable
}

func (forward *Forward) Add(cip string, conn net.Conn) {
	forward.table.Store(cip, conn)
}

func (forward *Forward) Get(cip string) (conn net.Conn) {
	val, ok := forward.table.Load(cip)
	if ok {
		return val.(net.Conn)
	}
	return nil
}

func (forward *Forward) Del(cip string) {
	forward.table.Delete(cip)
}

func (forward *Forward) Size() int {
	c := 0
	forward.table.Range(func(key, val interface{}) bool {
		c += 1
		return true
	})
	return c
}

func (forward *Forward) Broadcast(sndqueue chan *NotrClientContext, buff []byte) {
	forward.table.Range(func(key, val interface{}) bool {
		conn, ok := val.(net.Conn)
		if ok {
			bytes := common.Encode(common.C2C_DATA, buff)
			sndqueue <- &NotrClientContext{conn: conn, payload: bytes}
		}
		return true
	})
}

func (forward *Forward) Peer(sndqueue chan *NotrClientContext, dst string, buff []byte) error {
	c := forward.Get(dst)
	if c == nil {
		return fmt.Errorf("%s offline", dst)
	}

	bytes := common.Encode(common.C2C_DATA, buff)
	sndqueue <- &NotrClientContext{conn: c, payload: bytes}

	return nil
}
