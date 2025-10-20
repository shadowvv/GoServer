package tool

import (
	"sync"
)

type IdGenerator struct {
	lastTimestamp int64
	sequence      int64
	workerId      int64
	dataCenterId  int64
	lock          sync.Mutex
}

func NewIdGenerator(workerId, dataCenterId int64) *IdGenerator {
	return &IdGenerator{
		lastTimestamp: 0,
		sequence:      0,
		workerId:      workerId,
		dataCenterId:  dataCenterId,
	}
}

func (g *IdGenerator) NextId() int64 {
	g.lock.Lock()
	defer g.lock.Unlock()

	timestamp := GetCurrentTimeMillis()

	// 处理时间回拨
	if timestamp < g.lastTimestamp {
		// 等待直到时间追上
		timestamp = waitUntilNextMillis(g.lastTimestamp)
	}

	// 同一毫秒内增加序列号
	if timestamp == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & 4095
		if g.sequence == 0 {
			// 序列号用完，等待下一毫秒
			timestamp = waitUntilNextMillis(g.lastTimestamp)
		}
	} else {
		g.sequence = 0
	}

	g.lastTimestamp = timestamp
	return generateIdWithTime(g.workerId, g.dataCenterId, timestamp)
}

func generateIdWithTime(workerId, dataCenterId, timestamp int64) int64 {
	return ((timestamp - 1288834974657) << 22) | (dataCenterId << 17) | (workerId << 12) | (timestamp % 4096)
}

// waitUntilNextMillis 等待到下一毫秒
func waitUntilNextMillis(lastTimestamp int64) int64 {
	timestamp := GetCurrentTimeMillis()
	for timestamp <= lastTimestamp {
		timestamp = GetCurrentTimeMillis()
	}
	return timestamp
}
