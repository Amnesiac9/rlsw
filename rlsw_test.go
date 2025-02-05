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

		err = rl.WaitWithLimit(10 * time.Second)
		if err != nil {
			t.Error("Expected WaitWithLimit(10 *time.Second) to return no error: " + err.Error())
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
			return
		}

		rl.Clear()
		if rl.TimeStampCount() != 0 {
			t.Error("Expected TimeStampCount to be 0 after running Clear()")
			return
		}
	})

	t.Run("SetMaxWaitTime", func(t *testing.T) {

		testCases := []struct {
			name                string
			expectedMaxWaitTime time.Duration
		}{
			{name: "10 Minutes", expectedMaxWaitTime: 10 * time.Minute},
			{name: "20 Hours", expectedMaxWaitTime: 20 * time.Hour},
			{name: "20 Days", expectedMaxWaitTime: 20 * time.Hour},
			{name: "Zero", expectedMaxWaitTime: 0},
		}

		for _, test := range testCases {
			rl := NewRateLimiter(5, 10*time.Second)

			if rl.maxWaitTime != 0 {
				t.Error("Expected maxWaitTime to equal 0")
				return
			}

			rl.SetMaxWaitTime(test.expectedMaxWaitTime)

			if rl.maxWaitTime != test.expectedMaxWaitTime {
				t.Errorf("(%s) expected max wait time: %d got: %d", test.name, test.expectedMaxWaitTime, rl.maxWaitTime)
				return
			}
		}

	})

	t.Run("WaitWithInternalLimit", func(t *testing.T) {

		testCases := []struct {
			name          string
			maxWaitTime   time.Duration
			expectToError bool
		}{
			{name: "1 Second", maxWaitTime: 1 * time.Second, expectToError: true},
			{name: "5 Seconds", maxWaitTime: 5 * time.Second, expectToError: false},
		}

		for _, test := range testCases {
			rl := NewRateLimiter(1, 3*time.Second)

			if rl.maxWaitTime != 0 {
				t.Error("Expected maxWaitTime to equal 0")
				return
			}

			rl.SetMaxWaitTime(test.maxWaitTime)

			err := rl.WaitWithInternalLimit()
			if err != nil {
				t.Errorf("(%s) expected first WaitWithLimit() to return no error: %s", test.name, err.Error())
				return
			}

			err = rl.WaitWithInternalLimit()
			if test.expectToError && err == nil {
				t.Errorf("(%s) expected WaitWithInternalLimit() to error. Wait Time: %d, MaxWaitTime: %d", test.name, rl.GetWaitTime(), rl.maxWaitTime)
				return
			}
		}

	})

}
