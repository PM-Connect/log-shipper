package limiter

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type RateLimiter struct {
	Limit            uint64
	BaseKey          string
	Interval         time.Duration
	FlushInterval    time.Duration
	StoreMultiplier  int
	TotalStoredCount uint64

	syncedCount  uint64
	currentCount uint64
	currentKey   string

	ticker     *time.Ticker
	stopTicker chan bool

	Store Store
}

type StoreItem struct {
	Value uint64
	TTL   time.Time
}

type Store struct {
	sync.RWMutex
	Items map[string]StoreItem
}

func (s *Store) Get(key string) (StoreItem, bool) {
	s.RLock()
	defer s.RUnlock()

	value, ok := s.Items[key]
	return value, ok
}

func (s *Store) Set(key string, value StoreItem) {
	s.Lock()
	defer s.Unlock()
	s.Items[key] = value
}

func (s *Store) Len() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.Items)
}

func New(baseKey string, limit uint64, interval time.Duration, flushInterval time.Duration, storeMultiplier int) *RateLimiter {
	return &RateLimiter{
		Limit:           limit,
		BaseKey:         baseKey,
		Interval:        interval,
		FlushInterval:   flushInterval,
		StoreMultiplier: storeMultiplier,
		Store:           Store{Items: map[string]StoreItem{}},
		stopTicker:      make(chan bool),
	}
}

func (r *RateLimiter) updateCurrentKey() {
	curTime := time.Now()

	now := float64(curTime.UnixNano())

	ns := float64(r.Interval.Nanoseconds())

	currentTimeIntervalString := fmt.Sprintf("%d", uint64(math.Floor(now/ns)))

	r.currentKey = fmt.Sprintf("%s:%s", r.BaseKey, currentTimeIntervalString)

	r.Store.Set(r.currentKey, StoreItem{Value: 0, TTL: time.Now().Add(time.Duration(r.StoreMultiplier) * r.FlushInterval)})
}

func (r *RateLimiter) purgeExpiredKeys() {
	r.Store.Lock()
	for key, item := range r.Store.Items {
		if key != r.currentKey && time.Since(item.TTL) > 0 {
			atomic.AddUint64(&r.TotalStoredCount, ^uint64(item.Value-1))

			delete(r.Store.Items, key)
		}
	}
	r.Store.Unlock()
}

func (r *RateLimiter) Stop() {
	close(r.stopTicker)
	r.Flush()
}

func (r *RateLimiter) Flush() {
	flushCount := atomic.SwapUint64(&r.currentCount, 0)

	currentStored, ok := r.Store.Get(r.currentKey)
	if !ok {
		return
	}

	value := atomic.LoadUint64(&currentStored.Value) + flushCount
	atomic.StoreUint64(&currentStored.Value, value)

	r.Store.Set(r.currentKey, currentStored)

	atomic.AddUint64(&r.syncedCount, value)
	atomic.AddUint64(&r.TotalStoredCount, value)
}

func (r *RateLimiter) Increment(delta uint64) {
	atomic.AddUint64(&r.currentCount, delta)
}

func (r *RateLimiter) IsOverLimit() (uint64, bool) {
	rate := atomic.LoadUint64(&r.syncedCount) + atomic.LoadUint64(&r.currentCount)

	limit := atomic.LoadUint64(&r.Limit)

	if rate > limit {
		return rate - limit, true
	}

	return rate - limit, false
}

func (r *RateLimiter) IsAverageOverLimit() (uint64, uint64, bool) {
	length := r.Store.Len() + 1

	total := atomic.LoadUint64(&r.TotalStoredCount) + atomic.LoadUint64(&r.currentCount)

	mean := total / uint64(length)

	limit := atomic.LoadUint64(&r.Limit)

	if mean > limit {
		return mean - limit, mean, true
	}

	return mean - limit, mean, false
}

func (r *RateLimiter) Init() error {
	if r.Interval < time.Millisecond {
		return fmt.Errorf("minimum interval is 1 millisecond")
	}

	r.updateCurrentKey()

	r.ticker = time.NewTicker(r.FlushInterval)

	go func(r *RateLimiter) {
		for {
			select {
			case <-r.ticker.C:
				r.updateCurrentKey()
				r.purgeExpiredKeys()
				r.Flush()
			case <-r.stopTicker:
				r.ticker.Stop()
				return
			}
		}
	}(r)

	return nil
}
