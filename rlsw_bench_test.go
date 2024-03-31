package rlsw

import (
	"testing"
	"time"
)

// This is the fastest way, though it seems like it would create a lot of extra slices.
func (r *Limiter) clearExpiredOriginal(now time.Time) {
	for len(r.timestamps) > 0 && now.Sub(r.timestamps[0]) > r.window {
		r.timestamps = r.timestamps[1:]
	}
}

func (r *Limiter) clearExpiredIndexBased(now time.Time) {
	idx := 0
	for idx < len(r.timestamps) && now.Sub(r.timestamps[idx]) > r.window {
		idx++
	}
	r.timestamps = r.timestamps[idx:]
}

func (r *Limiter) clearExpiredBinary(now time.Time) {
	if len(r.timestamps) == 0 {
		return
	}

	// Binary search to find the index of the first non-expired timestamp
	start, end := 0, len(r.timestamps)-1
	for start <= end {
		mid := start + (end-start)/2
		if now.Sub(r.timestamps[mid]) > r.window {
			start = mid + 1
		} else {
			end = mid - 1
		}
	}

	// Remove expired timestamps
	if start > 0 {
		r.timestamps = r.timestamps[start:]
	}
}

func BenchmarkClearExpired(b *testing.B) {

	r := &Limiter{
		timestamps: make([]time.Time, 1000),
		window:     time.Minute,
	}
	now := time.Now()

	b.Run("Original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			r.clearExpiredOriginal(now)
		}
	})

	r.timestamps = make([]time.Time, 1000)

	b.Run("IndexBasedLoop", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			r.clearExpiredIndexBased(now)
		}
	})

	r.timestamps = make([]time.Time, 1000)

	b.Run("BinarySearch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			r.clearExpiredBinary(now)
		}
	})

}
