package rlsw

import (
	"fmt"
	"testing"
	"time"
)

func TestNewSubLimiter_PanicsOnInvalidInput(t *testing.T) {
	t.Run("PanicsWhenParentIsNil", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic when parent is nil, but did not panic")
			}
		}()
		NewSubLimiter(1, nil)
	})

	t.Run("PanicsWhenSubLimitExceedsParent", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic when subLimit > parent.limit, but did not panic")
			} else {
				msg := fmt.Sprintf("%v", r)
				expected := "subLimit (6) must not exceed parent limit (5)"
				if msg != expected {
					t.Errorf("Unexpected panic message: got %q, want %q", msg, expected)
				}
			}
		}()
		parent := NewRateLimiter(5, time.Second)
		NewSubLimiter(6, parent)
	})
}
func TestSubLimiter_Schedule_EnforcesHardLimit(t *testing.T) {
	parent := NewRateLimiter(5, time.Second)
	sub := NewSubLimiter(3, parent)

	// First 3 calls should pass with no wait
	for i := 0; i < 3; i++ {
		wait := sub.Schedule()
		if wait > 0 {
			t.Errorf("Expected 0 wait time on call %d, got %v", i+1, wait)
		}
	}

	// 4th call should exceed sub-limit of 3 and force a wait
	wait := sub.Schedule()
	if wait == 0 {
		t.Errorf("Expected non-zero wait time after sub-limit exceeded, got 0")
	}

	// Confirm the wait time is less than or equal to the window (1 second)
	if wait > time.Second {
		t.Errorf("Unexpectedly high wait time: got %v, expected ≤ 1s", wait)
	}

	// Optional: wait for timestamps to expire and confirm sub-limiter works again
	time.Sleep(wait + 10*time.Millisecond) // allow enough time for the oldest timestamp to expire

	// Should now succeed again
	wait = sub.Schedule()
	if wait > 0 {
		t.Errorf("Expected 0 wait time after window cleared, got %v", wait)
	}
}

func TestSubLimiter_Wait_EnforcesHardLimit(t *testing.T) {
	parent := NewRateLimiter(5, time.Second)
	sub := NewSubLimiter(3, parent)

	start := time.Now()

	// First 3 calls to Wait() should not block
	for i := 0; i < 3; i++ {
		before := time.Now()
		sub.Wait()
		elapsed := time.Since(before)

		if elapsed > 20*time.Millisecond {
			t.Errorf("Expected no significant wait on call %d, but got %v", i+1, elapsed)
		}
	}

	// 4th call should block (sub-limit exceeded)
	before := time.Now()
	sub.Wait() // This should block
	elapsed := time.Since(before)

	// Check that the wait time is reasonably close to the limiter window
	if elapsed < 900*time.Millisecond {
		t.Errorf("Expected Wait to block near 1s, got too short: %v", elapsed)
	}
	if elapsed > 1100*time.Millisecond {
		t.Errorf("Wait took too long: %v", elapsed)
	}

	// Ensure we can proceed again after the block
	before = time.Now()
	sub.Wait() // Should be fast now (timestamps expired)
	elapsed = time.Since(before)

	if elapsed > 20*time.Millisecond {
		t.Errorf("Expected fast wait after reset, but got %v", elapsed)
	}

	// Print helpful debug timing if needed
	t.Logf("Wait after sub-limit: %v", time.Since(start))
}

func TestSubLimiter_FullIntegration_WithAllow(t *testing.T) {
	parent := NewRateLimiter(5, time.Second)
	sub := NewSubLimiter(3, parent)

	// Start clean
	sub.Clear()
	if sub.TimeStampCount() != 0 {
		t.Fatalf("Expected timestamp count to be 0 after Clear, got %d", sub.TimeStampCount())
	}

	// Use Allow() up to sub-limit
	for i := 0; i < 3; i++ {
		allowed := sub.Allow()
		if !allowed {
			t.Errorf("Expected Allow() to return true on call %d, but got false", i+1)
		}
		if sub.GetWaitTime() > 0 && i != 2 {
			t.Errorf("Expected GetWaitTime to be 0 after Allow() under limit, got %v", sub.GetWaitTime())
		}
	}

	// 4th call to Allow should fail (sub-limit reached)
	if sub.Allow() {
		t.Errorf("Expected Allow() to return false after sub-limit reached, but got true")
	}

	// Confirm GetWaitTime is non-zero
	if wait := sub.GetWaitTime(); wait == 0 {
		t.Errorf("Expected GetWaitTime to be > 0 after hitting sub-limit")
	}

	// Confirm timestamp count is now 3
	if count := sub.TimeStampCount(); count != 3 {
		t.Errorf("Expected timestamp count to be 3 after Allow()s, got %d", count)
	}

	// Use Wait() to push another request (should block)
	start := time.Now()
	sub.Wait()
	elapsed := time.Since(start)
	if elapsed < 900*time.Millisecond || elapsed > 1100*time.Millisecond {
		t.Errorf("Expected Wait() to block near 1s, got %v", elapsed)
	}

	// Total timestamps should now be 1, since we waited the limit and added a new one after waiting.
	if count := sub.TimeStampCount(); count != 1 {
		t.Errorf("Expected timestamp count to be 1 after Wait(), got %d", count)
	}

	// Clear and ensure everything resets
	sub.Clear()
	if sub.TimeStampCount() != 0 {
		t.Errorf("Expected timestamp count to be 0 after Clear(), got %d", sub.TimeStampCount())
	}
	if sub.GetWaitTime() != 0 {
		t.Errorf("Expected GetWaitTime to be 0 after Clear(), got %v", sub.GetWaitTime())
	}

	// Ensure Allow() works again after Clear()
	if !sub.Allow() {
		t.Errorf("Expected Allow() to return true after Clear(), but got false")
	}
}

func TestSubLimiter_Schedule_RemovesOldestWhenParentAndSubLimitReached(t *testing.T) {
	parent := NewRateLimiter(3, time.Second)
	sub := NewSubLimiter(2, parent)

	// Fill sub-limiter to its limit
	for i := 0; i < 2; i++ {
		if wait := sub.Schedule(); wait > 0 {
			t.Errorf("Expected initial sub.Schedule() call %d to return 0 wait, got %v", i+1, wait)
		}
	}

	// Fill parent to its remaining limit using parent.Schedule()
	if wait := parent.Schedule(); wait > 0 {
		t.Errorf("Expected parent.Schedule() to fill final slot without wait, got %v", wait)
	}

	// At this point, parent has 3 timestamps, sub has 2 (all in parent.timestamps)
	if count := parent.TimeStampCount(); count != 3 {
		t.Fatalf("Expected parent timestamp count to be 3 before overflow test, got %d", count)
	}

	// This call should hit both limits and trigger parent.removeOldest()
	wait := sub.Schedule()
	if wait == 0 {
		t.Errorf("Expected sub.Schedule() to return non-zero wait when limits are exceeded")
	}

	// After scheduling, the count should stay at 3 due to removeOldest()
	if count := parent.TimeStampCount(); count != 3 {
		t.Errorf("Expected timestamp count to stay at 3 after Schedule(), got %d", count)
	}
}
