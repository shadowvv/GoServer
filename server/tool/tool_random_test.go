package tool

import (
	"sync"
	"testing"
)

// 运行多次，确保不会越界
const sampleTimes = 10000

// ---------- 基础功能测试 ----------

func TestRandIntRange(t *testing.T) {
	min, max := 1, 5

	for i := 0; i < sampleTimes; i++ {
		v := RandInt(min, max)
		if v < min || v > max {
			t.Fatalf("RandInt out of range: %d (min=%d max=%d)", v, min, max)
		}
	}
}

func TestRandIntSwapBounds(t *testing.T) {
	min, max := 10, 1

	for i := 0; i < sampleTimes; i++ {
		v := RandInt(min, max)
		if v < max || v > min { // swapped internally
			t.Fatalf("RandInt did not swap bounds correctly: %d", v)
		}
	}
}

func TestRandIntEqualBounds(t *testing.T) {
	min, max := 7, 7

	for i := 0; i < sampleTimes; i++ {
		v := RandInt(min, max)
		if v != min {
			t.Fatalf("RandInt should always return %d when min==max, got %d", min, v)
		}
	}
}

func TestRandInt32Range(t *testing.T) {
	var min int32 = -5
	var max int32 = 3

	for i := 0; i < sampleTimes; i++ {
		v := RandInt32(min, max)
		if v < min || v > max {
			t.Fatalf("RandInt32 out of range: %d (min=%d max=%d)", v, min, max)
		}
	}
}

// ---------- 确保 deterministic ----------

func TestNewRandomDeterministic(t *testing.T) {
	seed := int64(12345)

	r1 := NewRandomWithSeed(seed)
	r2 := NewRandomWithSeed(seed)

	// 前 N 次应完全一致，确保可复现
	for i := 0; i < 1000; i++ {
		if x, y := r1.Int(), r2.Int(); x != y {
			t.Fatalf("NewRandomWithSeed with same seed should be deterministic: %d != %d", x, y)
		}
	}
}

// ---------- 并发安全性测试（race 检测） ----------
// go test -race
func TestRandIntConcurrent(t *testing.T) {
	wg := sync.WaitGroup{}
	goroutines := 100
	callsEach := 10000

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < callsEach; j++ {
				_ = RandInt(-1000, 1000)
				_ = RandInt32(-1000, 1000)
			}
		}()
	}

	wg.Wait()
}

// ---------- Fuzz 测试（Go 1.18+） ----------

func FuzzRandInt(f *testing.F) {
	f.Add(1, 10)
	f.Add(-10, 10)
	f.Add(0, 0)
	f.Add(100, 1)

	f.Fuzz(func(t *testing.T, a int, b int) {
		v := RandInt(a, b)

		// normalize to know theoretical min/max
		min := a
		max := b
		if max < min {
			min, max = max, min
		}

		if v < min || v > max {
			t.Fatalf("RandInt fuzz out of range v=%d min=%d max=%d", v, min, max)
		}
	})
}

func FuzzRandInt32(f *testing.F) {
	f.Add(int32(-5), int32(5))

	f.Fuzz(func(t *testing.T, a int32, b int32) {
		v := RandInt32(a, b)

		min := a
		max := b
		if max < min {
			min, max = max, min
		}

		if v < min || v > max {
			t.Fatalf("RandInt32 fuzz out of range v=%d min=%d max=%d", v, min, max)
		}
	})
}

// ---------- 大量并发一致性行为验证 ----------

func TestRandIntStress(t *testing.T) {
	wg := sync.WaitGroup{}
	workers := 200
	perWorker := 5000

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < perWorker; j++ {
				v := RandInt(-1_000_000, 1_000_000)
				if v < -1_000_000 || v > 1_000_000 {
					t.Fatalf("stress RandInt out of range: %d", v)
				}
			}
		}()
	}

	wg.Wait()
}

// ---------- Benchmark 基准测试 ----------

func BenchmarkRandInt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = RandInt(-1000, 1000)
	}
}

func BenchmarkRandInt32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = RandInt32(-1000, 1000)
	}
}

func BenchmarkNewRandom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewRandomWithSeed(int64(i))
	}
}
