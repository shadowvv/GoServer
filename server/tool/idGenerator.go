package tool

import (
	"fmt"
	"sync"
)

const (
	EPOCH            = 1767225600000 // 2026-01-01
	TIME_STAMP_BITS  = 36            // 时间位数
	FUNCTION_ID_BITS = 6             // 功能位数
	WORKER_ID_BITS   = 6             // 节点id位数
	SEQUENCE_BITS    = 5             // 最大并发位数

	MAX_ID = 63
)

type IdGenerator struct {
	lastTimestamp int64 // 上一次生成ID的时间戳
	sequence      int64 // 当前毫秒内的序列号
	workerId      int64 // 工作节点ID
	functionId    int64 // 数据中心ID
	lock          sync.Mutex
}

func NewIdGenerator(workerId int64, functionId int64) *IdGenerator {
	if workerId == 0 {
		panic("[system] workerId must be greater than 0")
	}
	if workerId < 0 || workerId > MAX_ID {
		panic(fmt.Sprintf("[system] workerId must be between 0 and 63 currentWorkerId:%d", workerId))
	}
	if functionId < 0 || functionId > MAX_ID {
		panic(fmt.Sprintf("[system] functionId must be between 0 and 63 currentFunctionId:%d", functionId))
	}
	return &IdGenerator{
		lastTimestamp: 0,
		sequence:      0,
		workerId:      workerId,
		functionId:    functionId,
		lock:          sync.Mutex{},
	}
}

func (g *IdGenerator) NextId() int64 {
	g.lock.Lock()
	defer g.lock.Unlock()

	timestamp := UnixNowMilli()
	// 处理时间回拨
	if timestamp < g.lastTimestamp {
		// 等待直到时间追上
		timestamp = waitUntilNextMillis(g.lastTimestamp)
	}

	// 同一毫秒内增加序列号
	if timestamp == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & (int64(1<<SEQUENCE_BITS) - 1)
		if g.sequence == 0 {
			// 序列号用完，等待下一毫秒
			timestamp = waitUntilNextMillis(g.lastTimestamp)
		}
	} else {
		g.sequence = 0
	}

	g.lastTimestamp = timestamp
	return generateIdWithTime(g.workerId, g.functionId, timestamp, g.sequence)
}

// waitUntilNextMillis 等待到下一毫秒
func waitUntilNextMillis(lastTimestamp int64) int64 {
	timestamp := UnixNowMilli()
	for timestamp <= lastTimestamp {
		timestamp = UnixNowMilli()
	}
	return timestamp
}

func generateIdWithTime(workerId, functionId, timestamp, sequence int64) int64 {
	// 限制范围
	functionId &= (1<<FUNCTION_ID_BITS - 1)
	workerId &= (1<<WORKER_ID_BITS - 1)
	sequence &= (1<<SEQUENCE_BITS - 1)

	elapsed := timestamp - EPOCH
	elapsed &= (1<<TIME_STAMP_BITS - 1)

	id := (uint64(elapsed) << (FUNCTION_ID_BITS + WORKER_ID_BITS + SEQUENCE_BITS)) |
		(uint64(functionId) << (WORKER_ID_BITS + SEQUENCE_BITS)) |
		(uint64(workerId) << SEQUENCE_BITS) |
		uint64(sequence)

	return int64(id)
}
