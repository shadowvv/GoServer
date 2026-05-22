package etcd

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Config struct {
	Endpoints   []string      `yaml:"endpoints"`
	DialTimeout time.Duration `yaml:"dialTimeout"`
	TTL         int64         `yaml:"ttl"`
}

type EtcdService struct {
	Cli    *clientv3.Client
	Ctx    context.Context
	Cancel context.CancelFunc
}

func NewEtcdService(cfg *Config) (*EtcdService, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	})
	if err != nil {
		cancel()
		return nil, err
	}

	return &EtcdService{
		Cli:    cli,
		Ctx:    ctx,
		Cancel: cancel,
	}, nil
}

func (e *EtcdService) GetOnlineNodes(key string) (map[string]string, int64, error) {
	ctx, cancel := context.WithTimeout(e.Ctx, 3*time.Second)
	defer cancel()

	resp, err := e.Cli.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, 0, err
	}

	m := make(map[string]string)
	for _, kv := range resp.Kvs {
		m[string(kv.Key)] = string(kv.Value)
	}

	return m, resp.Header.Revision, nil
}

func (e *EtcdService) Close() error {
	e.Cancel()
	return e.Cli.Close()
}
