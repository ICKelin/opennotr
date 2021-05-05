package core

import (
	"fmt"
	"net"
	"syscall"
)

func checksum_add(buf []byte, seed uint32) uint32 {
	sum := seed
	for i, l := 0, len(buf); i < l; i += 2 {
		j := i + 1
		if j == l {
			sum += uint32(buf[i]) << 8
			break
		}
		sum += uint32(buf[i])<<8 | uint32(buf[j])
	}
	return sum
}

func checksum_warp(seed uint32) uint16 {
	sum := seed
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	csum := ^uint16(sum)

	// RFC 768
	if csum == 0 {
		csum = 0xffff
	}
	return csum
}

func CheckSum(buf []byte) uint16 {
	return checksum_warp(checksum_add(buf, 0))
}

func sendUDPViaRaw(fd int, src, dst *net.UDPAddr, payload []byte) error {
	iplen, ulen := uint16(28+len(payload)), uint16(8+len(payload))
	if iplen > 65535 {
		return fmt.Errorf("too big packet")
	}

	// UDP checksum: sip + dip + udp-head + payload + PROTO + ulen
	data := make([]byte, iplen)
	data[9] = syscall.IPPROTO_UDP
	copy(data[12:16], src.IP.To4())
	copy(data[16:20], dst.IP.To4())
	data[20] = byte(src.Port >> 8)
	data[21] = byte(src.Port)
	data[22] = byte(dst.Port >> 8)
	data[23] = byte(dst.Port)
	data[24] = byte(ulen >> 8)
	data[25] = byte(ulen)
	copy(data[28:], payload)

	uc := checksum_warp(checksum_add(data, uint32(ulen)))
	data[26] = byte(uc >> 8)
	data[27] = byte(uc)

	data[0] = 0x45
	data[2] = byte(iplen >> 8)
	data[3] = byte(iplen)
	data[6] = 0x40
	data[8] = 64
	ipc := CheckSum(data[:20])
	data[10] = byte(ipc >> 8)
	data[11] = byte(ipc)

	addr := syscall.SockaddrInet4{Port: dst.Port}
	copy(addr.Addr[:], data[16:20])
	return syscall.Sendto(fd, data, 0, &addr)
}
