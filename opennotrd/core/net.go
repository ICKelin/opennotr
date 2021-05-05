package core

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"github.com/ICKelin/opennotr/pkg/logs"
	"github.com/ICKelin/opennotr/pkg/proto"
)

func checksumAdd(buf []byte, seed uint32) uint32 {
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

func checksumWrapper(seed uint32) uint16 {
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
	return checksumWrapper(checksumAdd(buf, 0))
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

	uc := checksumWrapper(checksumAdd(data, uint32(ulen)))
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

func encode(raw []byte) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(raw)))
	buf = append(buf, raw...)
	return buf
}

func encodeProxyProtocol(protocol, sip, sport, dip, dport string) []byte {
	proxyProtocol := &proto.ProxyProtocol{
		Protocol: protocol,
		SrcIP:    sip,
		SrcPort:  sport,
		DstIP:    dip,
		DstPort:  dport,
	}

	body, _ := json.Marshal(proxyProtocol)
	return encode(body)
}

func getOriginDst(hdr []byte) (*net.UDPAddr, error) {
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
