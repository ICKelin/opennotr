package main

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	LocalIP       string          `json:"device_ip"`
	LocalListener string          `json:"listen"`
	Tap           bool            `json:"tap"`
	Clients       []*ClientConfig `json:"client"`
}

type ClientConfig struct {
	AuthKey string `json:"auth_key"`
	Domain  string `json:"domain"`
}

func ParseConfig(path string) (*Config, error) {
	cnt, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var conf Config
	err = json.Unmarshal(cnt, &conf)
	if err != nil {
		return nil, err
	}

	return &conf, nil
}
