package core

import (
	"net"
)

type mockUDPForward struct {
	*UDPForward
}

func (f *mockUDPForward) getOriginDst([]byte) (*net.UDPAddr, error) {
	return nil, nil
}
