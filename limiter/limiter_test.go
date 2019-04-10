package limiter

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var benchBy uint64
var benchOver bool

func benchmarkRateLimiter(loops int, delta int) {
	rl := New("limiter", 10000, time.Second, 1*time.Second, 10)

	_ = rl.Init()

	var by uint64
	var over bool

	for i := 0; i < loops; i++ {
		rl.Increment(uint64(delta))
		by, over = rl.IsOverLimit()
	}

	rl.Stop()

	benchBy = by
	benchOver = over
}

func TestRateLimiter(t *testing.T) {
	rl := New("limiter", 5, time.Second, 1*time.Second, 10)

	err := rl.Init()

	assert.Nil(t, err)

	numberOfOverages := 0

	for i := 0; i < 100; i++ {
		rl.Increment(1)

		_, over := rl.IsOverLimit()

		if over {
			numberOfOverages++
		}

		time.Sleep(10 * time.Millisecond)
	}

	rl.Stop()

	assert.Equal(t, 2, rl.Store.Len())
	assert.NotZero(t, numberOfOverages)
}

func TestRateLimiter_Average(t *testing.T) {
	rl := New("limiter", 2, 100*time.Millisecond, 100*time.Millisecond, 5)

	err := rl.Init()

	assert.Nil(t, err)

	timesOverAverage := 0

	for i := 0; i < 10; i++ {
		delta := 1

		if i < 3 {
			delta = 4
		} else {
			delta = 1
		}

		rl.Increment(uint64(delta))

		_, _, over := rl.IsAverageOverLimit()

		if over {
			timesOverAverage++
		}

		time.Sleep(100 * time.Millisecond)
	}

	rl.Stop()

	assert.NotZero(t, timesOverAverage)
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
