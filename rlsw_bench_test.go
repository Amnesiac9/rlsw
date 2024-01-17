package rlsw

import (
	"testing"
	"time"
)

func (r *Limiter) clearExpiredOriginal(now time.Time) {
	for len(r.timestamps) > 0 && now.Sub(r.timestamps[0]) > r.window {
		r.timestamps = r.timestamps[1:]
	}
}

func (r *Limiter) clearExpiredRevised(now time.Time) {
	idx := 0
	for idx < len(r.timestamps) && now.Sub(r.timestamps[idx]) > r.window {
		idx++
	}
	r.timestamps = r.timestamps[idx:]
}

func BenchmarkClearExpiredOriginal(b *testing.B) {
	r := &Limiter{
		timestamps: make([]time.Time, 1000),
		window:     time.Minute,
	}
	now := time.Now()

	for i := 0; i < b.N; i++ {
		r.clearExpiredOriginal(now)
	}
}

func BenchmarkClearExpiredRevised(b *testing.B) {
	r := &Limiter{
		timestamps: make([]time.Time, 1000),
		window:     time.Minute,
	}
	now := time.Now()

	for i := 0; i < b.N; i++ {
		r.clearExpiredRevised(now)
	}
}
