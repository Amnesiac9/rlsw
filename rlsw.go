package rlsw

import (
	"sync"
	"time"
)

// Limiter is a sliding window rate limiter that allows for a certain number of requests per duration.
type Limiter struct {
	mu         sync.Mutex
	timestamps []time.Time
	limit      int
	window     time.Duration
}

func NewRateLimiter(limit int, duration time.Duration) *Limiter {
	return &Limiter{
		timestamps: make([]time.Time, 0),
		limit:      limit,
		window:     duration,
	}
}

func (r *Limiter) Limit() int {
	return r.limit
}

func (r *Limiter) SetLimit(limit int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.limit = limit
}

func (r *Limiter) Window() time.Duration {
	return r.window
}

func (r *Limiter) SetWindow(window time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.window = window
}

// Clears the expired timestamps. Does not Lock or Unlock Mutex, never call on it's own.
func (r *Limiter) clearExpired(now time.Time) {
	for len(r.timestamps) > 0 && now.Sub(r.timestamps[0]) > r.window {
		r.timestamps = r.timestamps[1:]
	}
}

func (r *Limiter) addTime(timestamp time.Time) {
	r.timestamps = append(r.timestamps, timestamp)
}

// func (r *RateLimiterSW) waitTime() time.Duration {
// 	return r.window - time.Since(r.timestamps[0])
// }

func (r *Limiter) waitTime(now time.Time) time.Duration {
	return r.window - now.Sub(r.timestamps[0])
}

func (r *Limiter) removeOldest() {
	r.timestamps = r.timestamps[1:]
}

// Allow returns true if the window has space for another request and appends a timestamp to the window.
func (r *Limiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.clearExpired(now)

	if len(r.timestamps) >= r.limit {
		return false
	}

	r.addTime(now)
	return true
}

// Schedule() removes any expired timestamps, then returns the duration to wait before another request should be allowed.
//
// If the request is allowed, it will append the current timestamp to the window.
//
// If the request is not allowed, it will append the current timestamp + the wait time to the timestamps, then remove the oldest timestamp, even if it's not expired.
// This allows you to concurrently call Schedule() and ensure each request waits the appropriate amount of time.
func (r *Limiter) Schedule() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.clearExpired(now)

	if len(r.timestamps) >= r.limit {
		waitTime := r.waitTime(now)
		r.addTime(now.Add(waitTime)) // Append the timestamp with the future time that the wait time with expire at.
		r.removeOldest()             // Remove the oldest timestamp, this way, the next request will need to wait longer.
		return waitTime
	}

	r.addTime(now)
	return 0
}

// Wait calls time.Sleep(r.Schedule()). This blocks until the rate limiter allows another request. If blocked, it schedules the time in the future on the timestamps, and removes the oldest timestamp.
// This way, the next request will need to wait longer.
func (r *Limiter) Wait() {
	time.Sleep(r.Schedule())
}

// The problem with this is that if used with go routines, concurrent requests to GetWaitTime() will return the same or close to the wait WaitTime
// This won't be accurate if there is a time gap between the oldest time and the next available time.
func (r *Limiter) Wait_Old() {
	time.Sleep(r.GetWaitTime())
	r.addTime(time.Now())
}

// Clears expired timestamps, then gets the current wait time and returns it without appending to the timestamps. Returns 0 if there is no wait.
func (r *Limiter) GetWaitTime() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.clearExpired(now)

	if len(r.timestamps) >= r.limit {
		return r.waitTime(now)
	}

	return 0
}

// Clears any expired timestamps, then returns the current len of r.timestamps
func (r *Limiter) TimeStampCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clearExpired(time.Now())

	return len(r.timestamps)
}

// Returns a
// func SpaceBetween(r *RateLimiterSW) {

// }

// func GetRate(r *RateLimiterSW, requestCount int) {

// 	// get requests we can make per duration
// 	currentAmountWeCanMakeInWindow := r.limit - r.TimeStampCount()
// 	rpd := currentAmountWeCanMakeInWindow / r.Window().Seconds()
// }
