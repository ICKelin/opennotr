package main

import (
	"encoding/json"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	ServerAddr string      `yaml:"serverAddr"`
	Key        string      `yaml:"key"`
	Domain     string      `yaml:"domain"`
	HTTP       int         `yaml:"http"`
	HTTPS      int         `yaml:"https"`
	Grpc       int         `yaml:"grpc"`
	TCPs       map[int]int `yaml:"tcp"`
	UDPs       map[int]int `yaml:"udp"`
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
