package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pelletier/go-toml"
)

type Config struct {
	ServerConfig   ServerConfig   `toml:"server"`
	GatewayConfig  GatewayConfig  `toml:"gateway"`
	ProxyConfig    ProxyConfig    `toml:"proxy"`
	ResolverConfig ResolverConfig `toml:"resolver"`
}

type ServerConfig struct {
	ListenAddr string `toml:"listen"`  // server监听地址
	AuthKey    string `toml:"authKey"` // 鉴权key
	Domain     string `toml:"domain"`  // 域名后缀
}

type GatewayConfig struct {
	Cidr string `toml:"cidr"` // 网关配置
	IP   string `toml:"ip"`   // 网关ip
}

type ProxyConfig struct {
	RemoteAddr string `toml:"remoteAddr"`
}

type ResolverConfig struct {
	EtcdEndpoints []string `toml:"etcdEndpoints"`
}

func Parse(path string) (*Config, error) {
	cnt, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = toml.Unmarshal(cnt, &cfg)
	return &cfg, err
}

func (c *Config) String() string {
	cnt, _ := json.Marshal(c)
	return string(cnt)
}
