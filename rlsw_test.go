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
		{name: "Limit 10 | Window 1", limit: 10, window: 1},
		{name: "Limit 20 | Window 100ms", limit: 20, window: 100 * time.Millisecond},
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

}

//TODO: Write tests using go routines to check functionality of sliding window
