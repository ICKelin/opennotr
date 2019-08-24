package common

const (
	_ = iota

	// 心跳命令
	C2S_HEARTBEAT
	S2C_HEARTBEAT

	// 鉴权命令
	C2S_AUTHORIZE
	S2C_AUTHORIZE

	// 数据命令
	C2C_DATA
	C2S_DATA
	S2C_DATA
)

type C2SAuthorize struct {
	Key       string `json:"key"`
	HttpPort  int    `json:"http"`
	HttpsPort int    `json:"https"`
}

type S2CAuthorize struct {
	AccessIP string `json:"access_ip"`
	Gateway  string `json:"gateway"`
	Domain   string `json:"domain"`
}
