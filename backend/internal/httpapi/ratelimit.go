package httpapi

import (
	"sync"
	"time"
)

// rateLimiter is a simple in-memory fixed-window limiter keyed by client IP. It
// is per-process and resets on restart — adequate for throttling auth attempts,
// as the TRD's known-limitations note. No Redis.
type rateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	counters map[string]*window
}

type window struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, w time.Duration) *rateLimiter {
	return &rateLimiter{
		limit:    limit,
		window:   w,
		counters: make(map[string]*window),
	}
}

// allow records an attempt for ip and reports whether it is within the limit
// for the current window. The window resets lazily on the first call after it
// expires, so the map does not need a background sweeper for correctness.
func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	w := rl.counters[ip]
	if w == nil || now.After(w.reset) {
		rl.counters[ip] = &window{count: 1, reset: now.Add(rl.window)}
		return true
	}
	if w.count >= rl.limit {
		return false
	}
	w.count++
	return true
}
