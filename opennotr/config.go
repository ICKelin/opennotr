package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ICKelin/opennotr/internal/proto"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ServerAddr string              `yaml:"serverAddr"`
	Key        string              `yaml:"key"`
	Domain     string              `yaml:"domain"`
	Forwards   []proto.ForwardItem `yaml:"forwards"`
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
