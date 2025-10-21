package etcd

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdRegistry handles registration and watching.
type EtcdRegistry struct {
	client *clientv3.Client
}

func NewEtcdRegistry(endpoints []string, dialTimeout time.Duration) (*EtcdRegistry, error) {
	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	}
	c, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	return &EtcdRegistry{client: c}, nil
}

// RegisterService registers a service instance with a lease TTL.
// key: /services/<name>/<instanceID] value: address
func (r *EtcdRegistry) RegisterService(ctx context.Context, key, value string, ttlSec int64) (clientv3.LeaseID, error) {
	leaseResp, err := r.client.Grant(ctx, ttlSec)
	if err != nil {
		return 0, err
	}
	if _, err := r.client.Put(ctx, key, value, clientv3.WithLease(leaseResp.ID)); err != nil {
		return 0, err
	}
	// start auto keepalive
	ch, kaerr := r.client.KeepAlive(ctx, leaseResp.ID)
	if kaerr != nil {
		return 0, kaerr
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-ch:
				if !ok {
					// keepalive closed
					return
				}
			}
		}
	}()
	return leaseResp.ID, nil
}

// Unregister removes the key (and revokes lease if provided)
func (r *EtcdRegistry) Unregister(ctx context.Context, key string) error {
	_, err := r.client.Delete(ctx, key)
	return err
}

// WatchPrefix watches a prefix and sends events to a channel.
// Returns cancel function to stop watching.
func (r *EtcdRegistry) WatchPrefix(ctx context.Context, prefix string, ch chan<- clientv3.Event) (context.CancelFunc, error) {
	wctx, cancel := context.WithCancel(ctx)
	watchCh := r.client.Watch(wctx, prefix, clientv3.WithPrefix(), clientv3.WithPrevKV())
	go func() {
		for wresp := range watchCh {
			for _, ev := range wresp.Events {
				select {
				case ch <- *ev:
				case <-wctx.Done():
					return
				}
			}
		}
	}()
	return cancel, nil
}

// ListPrefix returns current key-values under prefix
func (r *EtcdRegistry) ListPrefix(ctx context.Context, prefix string) (map[string]string, error) {
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		m[string(kv.Key)] = string(kv.Value)
	}
	return m, nil
}
