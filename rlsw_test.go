package rlsw

import "testing"

func Test_RateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, 1)
	for i := 0; i < 10; i++ {
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
