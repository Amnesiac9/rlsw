package rlsw

import "time"

type RateLimiterInterface interface {
	Allow() bool
	Schedule() time.Duration
	Wait()
	GetWaitTime() time.Duration
	TimeStampCount() int
	Clear()
}

var _ RateLimiterInterface = (*Limiter)(nil)
var _ RateLimiterInterface = (*SubLimiter)(nil)
