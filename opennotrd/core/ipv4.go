package core

import "fmt"

type Packet []byte

func (p Packet) Invalid() bool {
	return len(p) < 20
}

func (p Packet) Version() int {
	return int((p[0] >> 4))
}

func (p Packet) Dst() string {
	return fmt.Sprintf("%d.%d.%d.%d", p[16], p[17], p[18], p[19])
}

func (p Packet) Src() string {
	return fmt.Sprintf("%d.%d.%d.%d", p[12], p[13], p[14], p[15])
}
