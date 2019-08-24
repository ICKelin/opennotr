package common

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

// 公共错误代码: (10000,20000)
const (
	CODE_SUCCESS, MSG_SUCCESS = 10000, "success"
)

// 公共错误结构
type NotrError struct {
	Code    int
	Message string
}

func NewNotrError(code int, message string) *NotrError {
	return &NotrError{
		Code:    code,
		Message: message,
	}
}

func (e *NotrError) Error() string {
	return fmt.Sprintf("ERROR %d: %s", e.Code, e.Message)
}

// 公共响应结构
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// 生成响应结构
func ObjBody(obj interface{}, err *NotrError) []byte {
	code := CODE_SUCCESS
	message := MSG_SUCCESS

	if err != nil {
		code = err.Code
		message = err.Message
	}

	body := &Body{
		Code:    code,
		Message: message,
		Data:    obj,
	}

	bytes, _ := json.Marshal(body)
	return bytes
}

// 根据响应结构生成数据对象
func BodyObj(bytes []byte, obj interface{}) error {
	body := Body{}
	err := json.Unmarshal(bytes, &body)
	if err != nil {
		return err
	}

	if body.Code != CODE_SUCCESS {
		return fmt.Errorf("ERROR %d:%s", body.Code, body.Message)
	}

	data, _ := json.Marshal(body.Data)
	err = json.Unmarshal(data, obj)
	return err
}

// 发送编码
func Encode(cmd byte, payload []byte) []byte {
	buff := make([]byte, 0)

	plen := make([]byte, 2)
	binary.BigEndian.PutUint16(plen, uint16(len(payload))+1)
	buff = append(buff, plen...)
	buff = append(buff, cmd)
	buff = append(buff, payload...)

	return buff
}

// 接收解码
func Decode(conn net.Conn) (byte, []byte, error) {
	plen := make([]byte, 2)
	conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	_, err := io.ReadFull(conn, plen)
	conn.SetReadDeadline(time.Time{})
	if err != nil {
		return 0, nil, err
	}

	payloadlength := binary.BigEndian.Uint16(plen)
	if payloadlength > 65535 {
		return 0, nil, fmt.Errorf("too big ippkt size %d", payloadlength)
	}

	resp := make([]byte, payloadlength)
	nr, err := io.ReadFull(conn, resp)
	if err != nil {
		return 0, nil, err
	}

	if nr < 1 {
		return 0, nil, fmt.Errorf("invalid pkt")
	}

	if nr != int(payloadlength) {
		return resp[0], resp[1:nr], fmt.Errorf("invalid payloadlength %d %d", nr, int(payloadlength))
	}

	return resp[0], resp[1:nr], nil
}
