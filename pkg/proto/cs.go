package proto

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
)

const (
	CmdAuth = iota
	CmdHeartbeat
	CmdData
)

type S2CHeartbeat struct {
}

type C2SHeartbeat struct{}

type C2SAuth struct {
	Key    string      `json:"key"`
	Domain string      `json:"domain"` // 域名，不传则由服务端随机生成
	HTTP   int         `json:"http"`   // 本地http端口，为0则不指定
	HTTPS  int         `json:"https"`  // http端口，为0则不指定
	Grpc   int         `json:"grpc"`   // grpc端口，为0则不指定
	TCPs   map[int]int `json:"tcps"`   // tcp端口
	UDPs   map[int]int `json:"udps"`   // udp端口
}

type S2CAuth struct {
	Domain  string `json:"domain"`  // 分配域名
	Vip     string `json:"vip"`     // 分配虚拟ip地址
	Gateway string `json:"gateway"` // 网关地址(cidr)
}

// 1字节版本
// 1字节命令
// 2字节长度
type Header [4]byte

func (h Header) Version() int {
	return int(h[0])
}

func (h Header) Cmd() int {
	return int(h[1])
}

func (h Header) Bodylen() int {
	return (int(h[2]) << 8) + int(h[3])
}

func Read(conn net.Conn) (Header, []byte, error) {
	h := Header{}
	_, err := io.ReadFull(conn, h[:])
	if err != nil {
		return h, nil, err
	}

	bodylen := h.Bodylen()
	if bodylen <= 0 {
		return h, nil, nil
	}

	body := make([]byte, bodylen)
	_, err = io.ReadFull(conn, body)
	if err != nil {
		return h, nil, err
	}

	return h, body, nil
}

func Write(conn net.Conn, cmd int, body []byte) error {
	bodylen := make([]byte, 2)
	binary.BigEndian.PutUint16(bodylen, uint16(len(body)))

	hdr := []byte{0x01, byte(cmd)}
	hdr = append(hdr, bodylen...)

	writebody := make([]byte, 0)
	writebody = append(writebody, hdr...)
	writebody = append(writebody, body...)
	_, err := conn.Write(writebody)
	return err
}

func WriteJSON(conn net.Conn, cmd int, obj interface{}) error {
	body, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return Write(conn, cmd, body)
}

func ReadJSON(conn net.Conn, obj interface{}) error {
	_, body, err := Read(conn)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, obj)
}
