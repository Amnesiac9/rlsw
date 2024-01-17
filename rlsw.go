package rlsw

import (
	"sync"
	"time"
)

// RateLimiterSW is a sliding window rate limiter that allows for a certain number of requests per duration.
type RateLimiterSW struct {
	mu         sync.Mutex
	timestamps []time.Time
	limit      int
	window     time.Duration
}

func NewRateLimiter(limit int, duration time.Duration) *RateLimiterSW {
	return &RateLimiterSW{
		timestamps: make([]time.Time, 0),
		limit:      limit,
		window:     duration,
	}
}

// Allow returns the duration to wait before another request should be allowed. If the duration is 0, then another request is allowed and the timestamp is recorded.
func (r *RateLimiterSW) AllowTime() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for len(r.timestamps) > 0 && now.Sub(r.timestamps[0]) > r.window {
		r.timestamps = r.timestamps[1:]
	}

	if len(r.timestamps) >= r.limit {
		return r.window - now.Sub(r.timestamps[0])
	}

	r.timestamps = append(r.timestamps, now)
	return 0
}

// Allow returns whether another request within the rate limit is allowed.
func (r *RateLimiterSW) Allow() bool {
	return r.AllowTime() == 0
}

// Wait blocks until the rate limiter allows another request.
// TODO: Does not add a timestamp if the request returns a time to sleep...
func (r *RateLimiterSW) Wait() {
	time.Sleep(r.AllowTime())
}
