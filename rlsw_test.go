package rlsw

import (
	"testing"
	"time"
)

func Test_RateLimiter(t *testing.T) {

	type testCase struct {
		name   string
		limit  int
		window time.Duration
	}

	testCases := []testCase{
		{name: "Limit 10 | Window 1", limit: 10, window: 400 * time.Millisecond},
		{name: "Limit 20 | Window 100ms", limit: 20, window: 500 * time.Millisecond},
	}

	for _, test := range testCases {
		rl := NewRateLimiter(1, 1)

		rl.SetLimit(test.limit)
		rl.SetWindow(test.window)

		if rl.GetLimit() != test.limit {
			t.Error("Expected rl.Limit() to equal test.limit")
		}

		if rl.GetWindow() != test.window {
			t.Error("Expected rl.Window() to equal test.window")
		}

		for i := 0; i < test.limit; i++ {
			if !rl.Allow() {
				t.Error("Expected Allow() to return true")
			}
		}
		if rl.Allow() {
			t.Error("Expected Allow() to return false")
		}

		rl.Wait()
		if !rl.Allow() {
			t.Error("Expected Allow() to return true")
		}

	}

	t.Run("WaitWithLimit", func(t *testing.T) {
		rl := NewRateLimiter(1, 3*time.Second)
		err := rl.WaitWithLimit(10 * time.Second)
		if err != nil {
			t.Error("Expected first WaitWithLimit() to return no error: " + err.Error())
		}

		err = rl.WaitWithLimit(1 * time.Second)
		if err == nil {
			t.Error("Expected WaitWithLimit(1 *time.Second) to error.")
		}

	})

	t.Run("TimeStampCount and Clear", func(t *testing.T) {
		rl := NewRateLimiter(5, 10*time.Second)
		for i := 1; i <= 5; i++ {
			rl.Allow()
		}
		if rl.TimeStampCount() != 5 {
			t.Error("Expected TimeStampCount to be 5 after running Allow() 5 times.")
		}

		rl.Clear()
		if rl.TimeStampCount() != 0 {
			t.Error("Expected TimeStampCount to be 0 after running Clear()")
		}
	})

}
