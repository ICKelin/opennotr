package core

import (
	"encoding/json"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	ServerConfig   ServerConfig      `yaml:"server"`
	DHCPConfig     DHCPConfig        `yaml:"dhcp"`
	ResolverConfig ResolverConfig    `yaml:"resolver"`
	Plugins        map[string]string `yaml:"plugin"`
}

type ServerConfig struct {
	ListenAddr string `yaml:"listen"`
	AuthKey    string `yaml:"authKey"`
	Domain     string `yaml:"domain"`
}

type DHCPConfig struct {
	Cidr string `yaml:"cidr"`
	IP   string `yaml:"ip"`
}

type ResolverConfig struct {
	EtcdEndpoints []string `yaml:"etcdEndpoints"`
}

func ParseConfig(path string) (*Config, error) {
	cnt, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(cnt, &cfg)
	return &cfg, err
}

func (c *Config) String() string {
	cnt, _ := json.Marshal(c)
	return string(cnt)
}
