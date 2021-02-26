package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pelletier/go-toml"
)

type Config struct {
	ServerAddr string `toml:"serverAddr"`
	Key        string `toml:"key"`
	Domain     string `toml:"domain"`
	HTTP       int    `toml:"http"`
	HTTPS      int    `toml:"https"`
	Grpc       int    `toml:"grpc"`
}

func ParseConfig(path string) (*Config, error) {
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
