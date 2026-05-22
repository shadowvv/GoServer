package etcd

import (
	"context"
	"fmt"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// LeaseHolder 表示一个节点注册信息和 lease
type LeaseHolder struct {
	mu sync.Mutex

	Key   string
	Value string
	TTL   int64

	LeaseID clientv3.LeaseID
	ctx     context.Context
	cancel  context.CancelFunc

	OnKeepAliveLost func()
	running         bool
}

// String 用于打印
func (l *LeaseHolder) String() string {
	return fmt.Sprintf("LeaseHolder{key=%s, value=%s, leaseID=%d}", l.Key, l.Value, l.LeaseID)
}

// RegisterNode 注册节点，并启动安全 keepAliveLoop
func (e *EtcdService) RegisterNode(key string, value string, ttl int64) (*LeaseHolder, error) {
	leaseResp, err := e.Cli.Grant(e.Ctx, ttl)
	if err != nil {
		return nil, err
	}
	_, err = e.Cli.Put(e.Ctx, key, value, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		return nil, err
	}
	// 创建可 cancel context，控制 keepAliveLoop 生命周期
	ctx, cancel := context.WithCancel(e.Ctx)
	holder := &LeaseHolder{
		Key:     key,
		Value:   value,
		TTL:     ttl,
		LeaseID: leaseResp.ID,
		ctx:     ctx,
		cancel:  cancel,
	}

	// 自动续约
	ch, err := e.Cli.KeepAlive(ctx, leaseResp.ID)
	if err != nil {
		cancel()
		return nil, err
	}

	// 默认 reRegister 回调
	holder.OnKeepAliveLost = func() {
		go e.safeReRegister(holder)
	}

	// 启动安全 keepAliveLoop
	go e.keepAliveLoop(holder, ch)
	return holder, nil
}

// safeReRegister 保证同一时间只有一个 reRegister
func (e *EtcdService) safeReRegister(holder *LeaseHolder) {
	holder.mu.Lock()
	if holder.running {
		holder.mu.Unlock()
		return
	}
	holder.running = true
	holder.mu.Unlock()

	defer func() {
		holder.mu.Lock()
		holder.running = false
		holder.mu.Unlock()
	}()

	for {
		newHolder, err := e.RegisterNode(holder.Key, holder.Value, holder.TTL)
		if err == nil {
			holder.mu.Lock()
			// 取消旧 holder context
			holder.cancel()
			// 替换 leaseID 和 cancel
			holder.LeaseID = newHolder.LeaseID
			holder.ctx = newHolder.ctx
			holder.cancel = newHolder.cancel
			holder.mu.Unlock()
			return
		}
		time.Sleep(3 * time.Second)
	}
}

// keepAliveLoop 安全处理 keepalive
func (e *EtcdService) keepAliveLoop(h *LeaseHolder, ch <-chan *clientv3.LeaseKeepAliveResponse) {
	for {
		select {
		case <-h.ctx.Done():
			return
		case ka, ok := <-ch:
			if !ok || ka == nil {
				if h.OnKeepAliveLost != nil {
					h.OnKeepAliveLost()
				}
				return
			}
			// 正常心跳，可用于监控
			// fmt.Println("[etcd] heartbeat", h.Key, "ttl", ka.TTL)
		}
	}
}

// UpdateNode 更新 key 的值
func (e *EtcdService) UpdateNode(key string, value string) error {
	_, err := e.Cli.Put(e.Ctx, key, value)
	return err
}

// UnregisterNode 注销节点，安全停止 keepAliveLoop
func (e *EtcdService) UnregisterNode(holder *LeaseHolder) error {
	holder.mu.Lock()
	holder.cancel() // 停掉 keepAliveLoop
	holder.mu.Unlock()

	_, _ = e.Cli.Delete(e.Ctx, holder.Key)
	_, _ = e.Cli.Revoke(e.Ctx, holder.LeaseID)
	return nil
}
