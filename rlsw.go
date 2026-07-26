package rlsw

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrWaitTimeExceedsLimit is returned when a request is refused because it would
// have had to wait longer than the caller was willing to.
var ErrWaitTimeExceedsLimit = errors.New("wait time exceeds limit")

// sleepOrCancel blocks for d, or until ctx is done, whichever happens first.
func sleepOrCancel(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}

	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Limiter is a sliding window rate limiter that allows for a certain number of requests per duration.
type Limiter struct {
	mu          sync.Mutex
	timestamps  []time.Time
	limit       int
	window      time.Duration
	maxWaitTime time.Duration
}

func NewRateLimiter(limit int, duration time.Duration) *Limiter {
	return &Limiter{
		timestamps: make([]time.Time, 0),
		limit:      limit,
		window:     duration,
	}
}

func (r *Limiter) GetLimit() int {
	return r.limit
}

func (r *Limiter) SetLimit(limit int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.limit = limit
}

func (r *Limiter) GetWindow() time.Duration {
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

func (r *Limiter) getWaitTime(now time.Time) time.Duration {
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
		waitTime := r.getWaitTime(now)
		r.addTime(now.Add(waitTime)) // Append the timestamp with the future time that the wait time will expire at.
		r.removeOldest()             // Remove the oldest timestamp, this way, the next request will need to wait longer.
		return waitTime
	}

	r.addTime(now)
	return 0
}

// ScheduleWithLimit behaves like Schedule, except it refuses to reserve a slot when the
// resulting wait would exceed max, returning ErrWaitTimeExceedsLimit instead.
//
// The check and the reservation happen under a single lock, so concurrent callers can
// never be handed a wait longer than the max they asked for. On rejection the window is
// left untouched: a caller that gives up must not consume capacity, otherwise it would
// push the wait out further for everyone behind it.
func (r *Limiter) ScheduleWithLimit(max time.Duration) (time.Duration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.clearExpired(now)

	if len(r.timestamps) >= r.limit {
		waitTime := r.getWaitTime(now)
		if waitTime > max {
			return 0, ErrWaitTimeExceedsLimit
		}
		r.addTime(now.Add(waitTime))
		r.removeOldest()
		return waitTime, nil
	}

	r.addTime(now)
	return 0, nil
}

// Wait calls time.Sleep(r.Schedule()). This blocks until the rate limiter allows another request. If blocked, it schedules the time in the future on the timestamps, and removes the oldest timestamp.
// This way, the next request will need to wait longer.
//
// Wait is unbounded: under sustained overload Schedule keeps pushing the window further
// into the future, so callers can end up sleeping for hours. Prefer WaitContext.
func (r *Limiter) Wait() {
	time.Sleep(r.Schedule())
}

// WaitContext reserves a slot and blocks until it is due, returning early if ctx is
// cancelled. If the required wait exceeds max it returns ErrWaitTimeExceedsLimit without
// blocking or reserving.
//
// Unlike Wait, this can never park a goroutine indefinitely.
func (r *Limiter) WaitContext(ctx context.Context, max time.Duration) error {
	waitTime, err := r.ScheduleWithLimit(max)
	if err != nil {
		return err
	}

	return sleepOrCancel(ctx, waitTime)
}

// The problem with this is that if used with go routines, concurrent requests to GetWaitTime() will return the same or close to the wait WaitTime
// This won't be accurate if there is a time gap between the oldest time and the next available time.
// func (r *Limiter) Wait_Old() {
// 	time.Sleep(r.GetWaitTime())
// 	r.addTime(time.Now())
// }

// Clears expired timestamps, then gets the current wait time and returns it without appending to the timestamps. Returns 0 if there is no wait.
func (r *Limiter) GetWaitTime() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.clearExpired(now)

	if len(r.timestamps) >= r.limit {
		return r.getWaitTime(now)
	}

	return 0
}

// Clears the expired timestamps. Uses a mutex to lock and unlock, safe to call manually.
//
// Not normally needed, since Allow(), Schedule(), and Wait() all clear the expired timestamps.
func (r *Limiter) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.timestamps = make([]time.Time, 0)
}

// Clears any expired timestamps, then returns the current len of r.timestamps
func (r *Limiter) TimeStampCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clearExpired(time.Now())

	return len(r.timestamps)
}

// WaitWithLimit blocks until the limiter allows another request, unless that would mean
// waiting longer than waitLimit, in which case it returns ErrWaitTimeExceedsLimit.
func (r *Limiter) WaitWithLimit(waitLimit time.Duration) error {
	waitTime, err := r.ScheduleWithLimit(waitLimit)
	if err != nil {
		return err
	}

	time.Sleep(waitTime)

	return nil
}

func (r *Limiter) SetMaxWaitTime(waitLimit time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.maxWaitTime = waitLimit
}

func (r *Limiter) WaitWithInternalLimit() error {
	r.mu.Lock()
	maxWaitTime := r.maxWaitTime
	r.mu.Unlock()

	if maxWaitTime == 0 {
		r.Wait()
		return nil
	}

	return r.WaitWithLimit(maxWaitTime)
}
