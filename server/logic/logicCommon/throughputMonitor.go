package logicCommon

import (
	"github.com/drop/GoServer/server/service/dbService"
	"time"
)

import (
	"context"
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"sync/atomic"
)

type ThroughputMonitor struct {
	redisKey      string
	writeInterval time.Duration
	ttl           time.Duration

	received int32
	handled  int32

	stopCh chan struct{}
}

func NewThroughputMonitor(redisKey string, writeInterval, ttl time.Duration) *ThroughputMonitor {
	m := &ThroughputMonitor{
		redisKey:      redisKey,
		writeInterval: writeInterval,
		ttl:           ttl,
		stopCh:        make(chan struct{}),
	}
	return m
}

// 累加接收数量
func (m *ThroughputMonitor) AddReceived(n int32) {
	atomic.AddInt32(&m.received, n)
}

// 累加处理数量
func (m *ThroughputMonitor) AddHandled(n int32) {
	atomic.AddInt32(&m.handled, n)
}

// 停止 monitor
func (m *ThroughputMonitor) Stop() {
	close(m.stopCh)
}

func (m *ThroughputMonitor) Start() {
	ticker := time.NewTicker(m.writeInterval)
	defer ticker.Stop()
	ctx := context.Background()

	for {
		select {
		case <-ticker.C:
			rec := atomic.SwapInt32(&m.received, 0)
			hand := atomic.SwapInt32(&m.handled, 0)
			val := fmt.Sprintf("%d:%d", hand, rec)

			go func(v string) {
				err := dbService.RDB.Set(ctx, m.redisKey, v, m.ttl).Err()
				if err != nil {
					logger.ErrorBySprintf("[ThroughputMonitor] write to redis failed key:%s err:%v", m.redisKey, err)
				}
			}(val)
		case <-m.stopCh:
			return
		}
	}
}
