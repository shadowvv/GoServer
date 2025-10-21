package rpc

import (
	"context"
	"fmt"
	"github.com/drop/GoServer/server/service/etcd"
	"github.com/drop/GoServer/server/tool"
	"time"

	"go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ServiceClientManager 管理某个服务的地址与连接池
type ServiceClientManager struct {
	serviceName string
	pool        *etcd.ConnPool
	logger      *zap.Logger
	cancelWatch context.CancelFunc
}

// NewServiceClientManager 创建管理器（会订阅etcd）
func NewServiceClientManager(ctx context.Context, logger *zap.Logger, registry *etcd.EtcdRegistry, serviceName string, dialOptions ...grpc.DialOption) (*ServiceClientManager, error) {
	mgr := &ServiceClientManager{
		serviceName: serviceName,
		pool: etcd.NewConnPool(func(addr string) (*grpc.ClientConn, error) {
			// use supplied dial options
			return grpc.Dial(addr, dialOptions...)
		}),
		logger: logger,
	}
	// initialize list
	prefix := fmt.Sprintf("/services/%s/", serviceName)
	list, err := registry.ListPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0, len(list))
	for _, v := range list {
		addrs = append(addrs, v)
	}
	mgr.pool.UpdateAddresses(addrs)

	// watch
	evCh := make(chan clientv3.Event, 16)
	wcancel, err := registry.WatchPrefix(ctx, prefix, evCh)
	if err != nil {
		return nil, err
	}
	mgr.cancelWatch = wcancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-evCh:
				_ = ev // handle add/remove: recalc addresses
				// list and update pool
				list, _ := registry.ListPrefix(context.Background(), prefix)
				addrs := make([]string, 0, len(list))
				for _, v := range list {
					addrs = append(addrs, v)
				}
				mgr.pool.UpdateAddresses(addrs)
				logger.Info("service endpoints updated", zap.String("service", serviceName), zap.Int("count", len(addrs)))
			}
		}
	}()
	return mgr, nil
}

// CallUnary 简化调用：传入一个 invoke 函数 (conn -> rpc call)
// 自动做重试（默认3次）, 超时通过 ctx 控制
func (m *ServiceClientManager) CallUnary(ctx context.Context, invoke func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error)) (interface{}, error) {
	// Try a few times with backoff and picking new connection each time
	var lastErr error
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		conn, err := m.pool.PickConn()
		if err != nil {
			lastErr = err
			time.Sleep(tool.Backoff(i)) // small backoff
			continue
		}
		// per-call timeout can be provided in ctx
		rsp, err := invoke(ctx, conn)
		if err == nil {
			return rsp, nil
		}
		// determine if retriable
		if !IsRetriableError(err) {
			return nil, err
		}
		lastErr = err
		m.logger.Warn("rpc call failed, retrying", zap.Error(err), zap.Int("retry", i))
		time.Sleep(tool.Backoff(i))
	}
	return nil, lastErr
}

// IsRetriableError - basic implementation, can be extended
func IsRetriableError(err error) bool {
	if err == context.DeadlineExceeded || err == context.Canceled {
		return false
	}
	// grpc status / transport errors are usually retriable
	return true
}
