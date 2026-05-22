package tool

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	randv2 "math/rand/v2"
	"time"
)

// 创建一个新的随机源,线程安全
func NewRandomWithSeed(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

func NewRandom() *randv2.Rand {
	return randv2.New(randv2.NewPCG(uint64(randomSeed()), uint64(randomSeed())))
}

// 全局随机源（默认使用 crypto/rand 作为种子）
var globalRand *randv2.Rand

func init() {
	globalRand = randv2.New(randv2.NewPCG(uint64(randomSeed()), uint64(randomSeed())))
}

// 生成随机种子（来自 crypto/rand）
func randomSeed() int64 {
	var b [8]byte
	if _, err := crand.Read(b[:]); err == nil {
		return int64(binary.LittleEndian.Uint64(b[:]))
	}
	return time.Now().UnixNano()
}

// RandInt 返回 [min, max] 的整数
func RandInt(min, max int) int {
	if max < min {
		min, max = max, min
	}
	if min == max {
		return min
	}
	return min + globalRand.IntN(max-min+1)
}

// RandInt32 返回 [min, max] 的 int32
func RandInt32(min, max int32) int32 {
	if max < min {
		min, max = max, min
	}
	if min == max {
		return min
	}
	return min + globalRand.Int32N(max-min+1)
}

// RandomWeight 随机返回 values[i]，权重为 weights[i]。出问题返回0
func RandomWeight(values []int32, weights []int32) int32 {
	totalWeight := int32(0)
	for _, weight := range weights {
		totalWeight += weight
	}
	randWeight := RandInt32(0, totalWeight-1)
	for i, weight := range weights {
		if randWeight < weight {
			return values[i]
		}
		randWeight -= weight
	}
	return 0
}
