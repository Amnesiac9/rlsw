package rlsw

import (
	"context"
	"time"
)

type RateLimiterInterface interface {
	Allow() bool
	Schedule() time.Duration
	ScheduleWithLimit(max time.Duration) (time.Duration, error)
	Wait()
	WaitWithLimit(waitLimit time.Duration) error
	WaitContext(ctx context.Context, max time.Duration) error
	GetWaitTime() time.Duration
	TimeStampCount() int
	Clear()
}

var _ RateLimiterInterface = (*Limiter)(nil)
var _ RateLimiterInterface = (*SubLimiter)(nil)
