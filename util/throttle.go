package util

import (
	"sync"
	"time"
)

// ThrottleLeading 立即執行，期間丟棄
func ThrottleLeading(f func(), interval time.Duration) func() {
	var mu sync.Mutex
	var last time.Time

	return func() {
		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		if last.IsZero() || now.Sub(last) >= interval {
			last = now
			f()
		}
	}
}

// ThrottleTrailing 窗口結束後執行最後一次
func ThrottleTrailing(f func(), interval time.Duration) func() {
	var mu sync.Mutex
	var timer *time.Timer

	return func() {
		mu.Lock()
		defer mu.Unlock()

		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(interval, func() {
			mu.Lock()
			defer mu.Unlock()
			f()
		})
	}
}

// ThrottleLeadingTrailing 立即執行，窗口結束補一次
func ThrottleLeadingTrailing(f func(), interval time.Duration) func() {
	var mu sync.Mutex
	var last time.Time
	var timer *time.Timer
	var pending bool

	return func() {
		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		if last.IsZero() || now.Sub(last) >= interval {
			// Leading edge
			last = now
			f()
		} else {
			// Trailing edge
			pending = true
			if timer != nil {
				timer.Stop()
			}
			wait := interval - now.Sub(last)
			timer = time.AfterFunc(wait, func() {
				mu.Lock()
				defer mu.Unlock()
				if pending {
					last = time.Now()
					f()
					pending = false
				}
			})
		}
	}
}
