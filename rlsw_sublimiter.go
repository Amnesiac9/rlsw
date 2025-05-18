package rlsw

import (
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

func (s *SubLimiter) Wait() {
	time.Sleep(s.Schedule())
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
