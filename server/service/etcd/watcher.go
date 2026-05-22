package etcd

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type WatchEvent struct {
	Key   string
	Value string
	Type  string // put / delete
	Lease int64
}

func (e *EtcdService) WatchNodes(key string, cb func(ev WatchEvent)) context.CancelFunc {
	ctx, cancel := context.WithCancel(e.Ctx)

	go func() {
		rch := e.Cli.Watch(ctx, key, clientv3.WithPrefix())
		for wresp := range rch {
			for _, ev := range wresp.Events {
				event := WatchEvent{
					Key:   string(ev.Kv.Key),
					Value: string(ev.Kv.Value),
					Lease: int64(ev.Kv.Lease),
				}
				switch ev.Type {
				case clientv3.EventTypePut:
					event.Type = "put"
				case clientv3.EventTypeDelete:
					event.Type = "delete"
				}
				cb(event)
			}
		}
	}()

	return cancel
}
