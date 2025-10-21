package tool

import (
	"math"
	"math/rand"
	"time"
)

// Backoff compute exponential backoff with jitter
func Backoff(attempt int) time.Duration {
	base := 50 * time.Millisecond
	max := 2 * time.Second
	jitter := time.Duration(rand.Int63n(int64(base)))
	exp := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
	d := exp + jitter
	if d > max {
		return max
	}
	return d
}
