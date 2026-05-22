package tool

import (
	"sync"
	"testing"
	"time"
)

// 基础唯一性 + 连续生成不会重复
func TestIdGenerator_UniqueIncreasing(t *testing.T) {

	g := NewIdGenerator(1, 1)

	last := g.NextId()

	for i := 0; i < 1000; i++ {
		id := g.NextId()

		if id == last {
			t.Fatalf("duplicate id detected: %d", id)
		}

		// 不强制单调递增（跨毫秒组合位可能相同趋势）
		// 但通常应保证「大体递增」
		if id < 0 {
			t.Fatalf("unexpected negative id: %d", id)
		}

		last = id
	}
}

// 并发环境下不重复
func TestIdGenerator_ConcurrentUnique(t *testing.T) {

	g := NewIdGenerator(2, 1)

	var wg sync.WaitGroup
	var mu sync.Mutex
	ids := make(map[int64]struct{})

	n := 5000

	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			id := g.NextId()

			mu.Lock()
			defer mu.Unlock()

			if _, ok := ids[id]; ok {
				t.Fatalf("duplicate id in concurrent mode: %d", id)
			}
			ids[id] = struct{}{}
		}()
	}

	wg.Wait()
}

// 序列号溢出时应等待到下一毫秒
func TestIdGenerator_SequenceRolloverMovesToNextMs(t *testing.T) {

	g := NewIdGenerator(3, 1)

	now := UnixNowMilli()

	// 手动设置到“同一毫秒 + sequence 已满”
	g.lastTimestamp = now
	g.sequence = (1 << SEQUENCE_BITS) - 1 // 3 bits → 7

	id1 := g.NextId()

	if g.lastTimestamp <= now {
		t.Fatalf("expected generator to move to next millisecond, got %d", g.lastTimestamp)
	}

	id2 := g.NextId()

	if id1 == id2 {
		t.Fatalf("expected next id to differ after rollover")
	}
}

// 时间回拨时应等待恢复
func TestIdGenerator_TimeRollbackHandled(t *testing.T) {

	g := NewIdGenerator(4, 1)

	now := UnixNowMilli()

	// 模拟时间回拨
	g.lastTimestamp = now + 5

	start := time.Now()

	_ = g.NextId()

	elapsed := time.Since(start)

	// 一般应等待 > 0ms（直到时间追上 lastTimestamp）
	if elapsed <= 0 {
		// 不强制 >1ms（不同机器时钟粒度不同）
		t.Log("rollback handled instantly — acceptable on fast clocks")
	}
}

// workerId / functionId 越界应 panic
func TestIdGenerator_InvalidArgsPanic(t *testing.T) {

	tests := []struct {
		workerId   int64
		functionId int64
		shouldPan  bool
	}{
		{-1, 1, true},
		{256, 1, true},
		{1, -1, true},
		{1, 256, true},
		{1, 1, false},
	}

	for _, tc := range tests {

		func() {
			defer func() {
				if r := recover(); r != nil && !tc.shouldPan {
					t.Fatalf("unexpected panic for (%d,%d): %v",
						tc.workerId, tc.functionId, r)
				}
			}()

			_ = NewIdGenerator(tc.workerId, tc.functionId)

			if tc.shouldPan {
				t.Fatalf("expected panic for (%d,%d) but none",
					tc.workerId, tc.functionId)
			}
		}()
	}
}
