package rlsw

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// A refused request must not reserve a slot. If it did, callers that gave up would still
// push the window further into the future for everyone queued behind them.
func Test_ScheduleWithLimit_RejectionDoesNotReserve(t *testing.T) {
	limiters := map[string]RateLimiterInterface{
		"Limiter":    NewRateLimiter(1, time.Minute),
		"SubLimiter": NewSubLimiter(1, NewRateLimiter(1, time.Minute)),
	}

	for name, rl := range limiters {
		t.Run(name, func(t *testing.T) {
			// Consume the only slot so the next caller has to wait.
			if _, err := rl.ScheduleWithLimit(time.Minute); err != nil {
				t.Fatalf("first call should be allowed immediately, got: %v", err)
			}

			before := rl.GetWaitTime()

			// Refuse a caller that is not willing to wait.
			if _, err := rl.ScheduleWithLimit(time.Nanosecond); !errors.Is(err, ErrWaitTimeExceedsLimit) {
				t.Fatalf("expected ErrWaitTimeExceedsLimit, got: %v", err)
			}

			// The wait only ever shrinks as time passes; it must not have jumped forward.
			if after := rl.GetWaitTime(); after > before {
				t.Errorf("refused request extended the window: wait went from %v to %v", before, after)
			}
		})
	}
}

// No caller may block for longer than the max it asked for. The old implementation checked
// GetWaitTime and then called Wait under a separate lock, so a concurrent caller could slip
// in between and leave this one sleeping far longer than it agreed to.
func Test_WaitContext_NeverExceedsMaxUnderConcurrency(t *testing.T) {
	const (
		callers = 50
		max     = 200 * time.Millisecond
		slack   = 250 * time.Millisecond
	)

	rl := NewRateLimiter(5, 500*time.Millisecond)

	var overruns, admitted atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			start := time.Now()
			err := rl.WaitContext(context.Background(), max)
			elapsed := time.Since(start)

			// Refusals are fine and expected; admissions that overshot are not.
			if err == nil {
				admitted.Add(1)
				if elapsed > max+slack {
					overruns.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	// Guard against the assertion below passing simply because everyone was refused.
	if admitted.Load() == 0 {
		t.Fatal("no caller was admitted, the overrun check would be vacuous")
	}
	if n := overruns.Load(); n != 0 {
		t.Errorf("%d of %d admitted callers blocked longer than max (%v)", n, admitted.Load(), max)
	}
}

// A cancelled request must unwind immediately instead of sleeping out its reservation.
func Test_WaitContext_ReturnsOnCancel(t *testing.T) {
	rl := NewRateLimiter(1, 5*time.Second)

	// Consume the slot so the next caller is forced to wait ~5s.
	if _, err := rl.ScheduleWithLimit(5 * time.Second); err != nil {
		t.Fatalf("first call should be allowed immediately, got: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	start := time.Now()
	err := rl.WaitContext(ctx, 10*time.Second)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
	if elapsed > time.Second {
		t.Errorf("cancellation took %v to unwind, expected to return promptly", elapsed)
	}
}

// An uncontended caller is admitted without waiting and without error.
func Test_WaitContext_AllowsWhenUncontended(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	for i := 0; i < 2; i++ {
		if err := rl.WaitContext(context.Background(), time.Millisecond); err != nil {
			t.Fatalf("call %d should have been admitted immediately, got: %v", i, err)
		}
	}

	// The window is now full, so an impatient caller is refused rather than parked.
	if err := rl.WaitContext(context.Background(), time.Nanosecond); !errors.Is(err, ErrWaitTimeExceedsLimit) {
		t.Errorf("expected ErrWaitTimeExceedsLimit once saturated, got: %v", err)
	}
}
