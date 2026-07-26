package rlsw

import (
	"context"
	"fmt"
	"time"
)

type SubLimiter struct {
	parent *Limiter
	limit  int
}

// SubLimiter spins off a sub quantity of the parent ratelimiter to allow a smaller reserved amount.
// If any parent limiter or sublimiter fills the parent's timestamps to the subLimit, even if the sublimiter did not use them, the sublimiter will yield until it's lower limit becomes available.
// Be sure that the limit here is lower than the parent's limit, otherwise this function will panic.
func NewSubLimiter(subLimit int, parent *Limiter) *SubLimiter {
	if parent == nil {
		panic("parent limiter cannot be nil")
	}
	if parent.limit < subLimit {
		panic(fmt.Sprintf("subLimit (%d) must not exceed parent limit (%d)", subLimit, parent.limit))
	}
	return &SubLimiter{
		parent: parent,
		limit:  subLimit,
	}
}

func (s *SubLimiter) Allow() bool {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	now := time.Now()
	s.parent.clearExpired(now)

	if len(s.parent.timestamps) >= s.limit {
		return false
	}

	s.parent.addTime(now)
	return true
}

func (s *SubLimiter) Schedule() time.Duration {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	now := time.Now()
	s.parent.clearExpired(now)

	// If sub-limiter has hit its limit, schedule a future request
	if len(s.parent.timestamps) >= s.limit {
		waitTime := s.parent.getWaitTime(now)

		// Schedule the request for the future
		s.parent.addTime(now.Add(waitTime))

		// Only remove a timestamp if parent has also hit its own limit
		// This prevents exceeding the global request capacity
		// And allows the next request to be forced to wait for the next available time.
		if len(s.parent.timestamps) >= s.parent.limit {
			s.parent.removeOldest()
		}
		return waitTime
	}

	s.parent.addTime(now)
	return 0
}

// ScheduleWithLimit behaves like Schedule, except it refuses to reserve a slot when the
// resulting wait would exceed max, returning ErrWaitTimeExceedsLimit instead.
//
// The check and the reservation happen under a single lock, so concurrent callers can
// never be handed a wait longer than the max they asked for. On rejection the parent
// window is left untouched.
func (s *SubLimiter) ScheduleWithLimit(max time.Duration) (time.Duration, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	now := time.Now()
	s.parent.clearExpired(now)

	if len(s.parent.timestamps) >= s.limit {
		waitTime := s.parent.getWaitTime(now)
		if waitTime > max {
			return 0, ErrWaitTimeExceedsLimit
		}

		s.parent.addTime(now.Add(waitTime))

		// Only remove a timestamp if parent has also hit its own limit, matching Schedule.
		if len(s.parent.timestamps) >= s.parent.limit {
			s.parent.removeOldest()
		}
		return waitTime, nil
	}

	s.parent.addTime(now)
	return 0, nil
}

// Wait is unbounded: under sustained overload Schedule keeps pushing the window further
// into the future, so callers can end up sleeping for hours. Prefer WaitContext.
func (s *SubLimiter) Wait() {
	time.Sleep(s.Schedule())
}

// WaitWithLimit blocks until the limiter allows another request, unless that would mean
// waiting longer than waitLimit, in which case it returns ErrWaitTimeExceedsLimit.
func (s *SubLimiter) WaitWithLimit(waitLimit time.Duration) error {
	waitTime, err := s.ScheduleWithLimit(waitLimit)
	if err != nil {
		return err
	}

	time.Sleep(waitTime)

	return nil
}

// WaitContext reserves a slot and blocks until it is due, returning early if ctx is
// cancelled. If the required wait exceeds max it returns ErrWaitTimeExceedsLimit without
// blocking or reserving.
func (s *SubLimiter) WaitContext(ctx context.Context, max time.Duration) error {
	waitTime, err := s.ScheduleWithLimit(max)
	if err != nil {
		return err
	}

	return sleepOrCancel(ctx, waitTime)
}

// Clears expired timestamps, then gets the current wait time and returns it without appending to the timestamps. Returns 0 if there is no wait.
func (s *SubLimiter) GetWaitTime() time.Duration {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	now := time.Now()
	s.parent.clearExpired(now)

	if len(s.parent.timestamps) >= s.limit {
		return s.parent.getWaitTime(now)
	}

	return 0
}

// Clears any expired timestamps, then returns the current len of r.timestamps
func (r *SubLimiter) TimeStampCount() int {
	return r.parent.TimeStampCount()
}

// Clears the expired timestamps. Uses a mutex to lock and unlock, safe to call manually.
//
// Not normally needed, since Allow(), Schedule(), and Wait() all clear the expired timestamps.
func (r *SubLimiter) Clear() {
	r.parent.Clear()
}
