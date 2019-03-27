package limiter

import (
	"fmt"
	"math"
	"sync/atomic"
	"time"
)

type RateLimiter struct {
	Limit uint64
	BaseKey string
	Interval time.Duration
	FlushInterval time.Duration

	syncedCount uint64
	currentCount uint64
	currentKey string

	ticker *time.Ticker
	stopTicker chan bool

	Store map[string]StoreItem
}

type StoreItem struct {
	Value uint64
	Time time.Time
}

func New(baseKey string, limit uint64, interval time.Duration, flushInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		Limit: limit,
		BaseKey: baseKey,
		Interval: interval,
		FlushInterval: flushInterval,
		Store: map[string]StoreItem{},
	}
}

func (r *RateLimiter) updateCurrentKey() {
	curTime := time.Now()

	now := float64(curTime.UnixNano())

	ns := float64(r.Interval.Nanoseconds())

	currentTimeIntervalString := fmt.Sprintf("%d", int64(math.Floor(now/ns)))

	r.currentKey = fmt.Sprintf("%s:%s", r.BaseKey, currentTimeIntervalString)

	r.Store[r.currentKey] = StoreItem{Value: 0, Time: curTime}

	for key, item := range r.Store {
		if key != r.currentKey && time.Since(item.Time) > r.FlushInterval {
			delete(r.Store, key)
		}
	}
}

func (r *RateLimiter) Stop() {
	close(r.stopTicker)
	r.Flush()
}

func (r *RateLimiter) Flush() {
	flushCount := atomic.SwapUint64(&r.currentCount, 0)

	currentStored := r.Store[r.currentKey]
	currentStored.Value += flushCount
	r.Store[r.currentKey] = currentStored

	r.syncedCount = r.Store[r.currentKey].Value
}

func (r *RateLimiter) Increment(delta uint64) {
	atomic.AddUint64(&r.currentCount, delta)
}

func (r *RateLimiter) IsOverLimit() (int64, bool) {
	rate := r.syncedCount + r.currentCount
	if rate > r.Limit {
		return int64(rate) - int64(r.Limit), true
	}

	return int64(rate) - int64(r.Limit), false
}

func (r *RateLimiter) Init() error {
	if r.Interval < time.Second {
		return fmt.Errorf("minimum interval is 1 second")
	}

	r.updateCurrentKey()

	r.ticker = time.NewTicker(r.FlushInterval)

	go func(r *RateLimiter) {
		for {
			select {
			case <- r.ticker.C:
				r.updateCurrentKey()
				r.Flush()
			case <-r.stopTicker:
				r.ticker.Stop()
				return
			}
		}
	}(r)

	return nil
}
