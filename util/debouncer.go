package util

import (
	"sync"
	"time"
)

type Debouncer interface {
	Trigger()
	RunNow()
	Stop()
}

func NewDebouncer(fn func(), delay time.Duration) Debouncer {
	return &debouncer{
		fn:    fn,
		delay: delay,
	}
}

type debouncer struct {
	fn    func()
	delay time.Duration
	timer *time.Timer
	done  chan bool
	mu    sync.Mutex
}

func (d *debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer == nil {
		d.done = make(chan bool)
		d.timer = time.NewTimer(d.delay)
		go func() {
			for {
				select {
				case <-d.timer.C:
					d.fn()
				case <-d.done:
					return
				}
			}
		}()
	} else {
		if !d.timer.Stop() {
			select {
			case <-d.timer.C:
			default:
			}
		}
		d.timer.Reset(d.delay)
	}
}

func (d *debouncer) RunNow() {
	d.fn()
}

func (d *debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		if !d.timer.Stop() {
			select {
			case <-d.timer.C:
			default:
			}
		}
		close(d.done)
		d.timer = nil
		d.done = nil
	}
}
