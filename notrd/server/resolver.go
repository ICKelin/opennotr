package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"
)

type record struct {
	Host string `json:"host"`
}

type Resolver struct {
	endpoints []string
	cli       *clientv3.Client
}

func NewResolve(endpoints []string) (*Resolver, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Minute * 1,
	})
	if err != nil {
		return nil, err
	}

	return &Resolver{
		endpoints: endpoints,
		cli:       cli,
	}, nil
}

func (r *Resolver) ApplyDomain(domain, ip string) error {
	sp := strings.Split(domain, ".")
	if len(sp) == 0 {
		return fmt.Errorf("invalid domain: %s", domain)
	}

	key := "/skydns"
	for i := len(sp) - 1; i >= 0; i-- {
		key = fmt.Sprintf("%s/%s", key, sp[i])
	}

	value := &record{Host: ip}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = r.cli.Put(context.Background(), key, string(b))
	return err
}
