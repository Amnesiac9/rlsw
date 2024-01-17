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

func (r *RateLimiterSW) Limit() int {
	return r.limit
}

func (r *RateLimiterSW) SetLimit(limit int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.limit = limit
}

func (r *RateLimiterSW) Window() time.Duration {
	return r.window
}

func (r *RateLimiterSW) SetWindow(window time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.window = window
}

// Allow returns true if the window has space for another request and appends a timestamp to the window.
func (r *RateLimiterSW) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	// Clear expired
	// TODO: Could make this more efficient by getting the index of the maximum timestamps to clear?
	for len(r.timestamps) > 0 && now.Sub(r.timestamps[0]) > r.window {
		r.timestamps = r.timestamps[1:]
	}

	if len(r.timestamps) >= r.limit {
		return false
	}

	r.timestamps = append(r.timestamps, now)
	return true
}

// Allow returns the duration to wait before another request should be allowed. If the duration is 0, then 0 is returned and the timestamp is recorded.
func (r *RateLimiterSW) AllowTime() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for len(r.timestamps) > 0 && now.Sub(r.timestamps[0]) > r.window {
		r.timestamps = r.timestamps[1:]
	}

	if len(r.timestamps) >= r.limit {
		waitTime := r.window - now.Sub(r.timestamps[0])
		r.timestamps = append(r.timestamps, now.Add(waitTime)) // Append the timestamp with the future time that needs to be waited.
		r.timestamps = r.timestamps[1:]                        // Remove the oldest timestamp, this way, the next request will need to wait longer.
		return waitTime
	}

	r.timestamps = append(r.timestamps, now)
	return 0
}

// Wait blocks until the rate limiter allows another request. If blocked, it schedules the time in the future on the timestamps, and removes the oldest timestamp.
// This way, the next request will need to wait longer.
func (r *RateLimiterSW) Wait() {
	time.Sleep(r.AllowTime())
}

//// WIP

// Gets the current wait time and returns it without appending to the requests. Returns 0 if there is no wait.
func (r *RateLimiterSW) WaitTime() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for len(r.timestamps) > 0 && now.Sub(r.timestamps[0]) > r.window {
		r.timestamps = r.timestamps[1:]
	}

	if len(r.timestamps) >= r.limit {
		return r.window - now.Sub(r.timestamps[0])
	}

	return 0
}

// Add the current time to the timestamps of the RateLimiter.
// func (r *RateLimiterSW) AddNow() {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
// 	r.timestamps = append(r.timestamps, time.Now())
// }

// Clears the expired timestamps. Does not Lock or Unlock Mutex, never call on it's own.
// func (r *RateLimiterSW) clearExpired(now time.Time) {
// 	for len(r.timestamps) > 0 && now.Sub(r.timestamps[0]) > r.window {
// 		r.timestamps = r.timestamps[1:]
// 	}
// }
