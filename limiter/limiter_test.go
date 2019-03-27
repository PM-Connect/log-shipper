package limiter

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var benchBy int64
var benchOver bool

func benchmarkRateLimiter(loops int, delta int) {
	rl := New("limiter", 10000, time.Second, 1 * time.Second)

	_ = rl.Init()

	var by int64
	var over bool

	for i := 0; i < loops; i++ {
		rl.Increment(uint64(delta))
		by, over = rl.IsOverLimit()
	}

	benchBy = by
	benchOver = over
}

func TestRateLimiter(t *testing.T) {
	rl := New("limiter", 5, time.Second, 1 * time.Second)

	err := rl.Init()

	assert.Nil(t, err)

	for i := 0; i < 100; i++ {
		rl.Increment(1)

		_, over := rl.IsOverLimit()

		if i < 5 {
			assert.False(t, over)
		} else {
			assert.True(t, over)
		}

		time.Sleep(20 * time.Millisecond)
	}
}

func BenchmarkRateLimiter1000000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchmarkRateLimiter(1000000, 1)
	}
}

func BenchmarkRateLimiter10000000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchmarkRateLimiter(10000000, 1)
	}
}

func BenchmarkRateLimiter100000000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchmarkRateLimiter(100000000, 1)
	}
}

func BenchmarkRateLimiter1000000x100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchmarkRateLimiter(1000000, 100)
	}
}

func BenchmarkRateLimiter10000000x100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchmarkRateLimiter(10000000, 100)
	}
}

func BenchmarkRateLimiter100000000x100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchmarkRateLimiter(100000000, 100)
	}
}
