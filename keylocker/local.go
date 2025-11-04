package keylocker

import (
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

func Local(key string) Locker {
	val, _ := localLockers.LoadOrStore(key, sync.OnceValue(func() *localItem {
		return &localItem{
			Cond:     sync.NewCond(&sync.Mutex{}),
			key:      key,
			lastTime: time.Now(),
		}
	}))
	get := val.(func() *localItem)
	startAutoClear()
	return get()
}

var localLockers sync.Map

type localItem struct {
	waiting  int
	locked   bool
	lastTime time.Time
	*sync.Cond
	key string
}

func (l *localItem) Lock() {
	l.L.Lock()
	defer l.L.Unlock()
	for l.locked {
		l.waiting++
		l.Wait()
		l.waiting--
	}
	l.locked = true
}

func (l *localItem) TryLock() bool {
	l.L.Lock()
	defer l.L.Unlock()
	if l.locked {
		return false
	}

	l.locked = true
	return true
}

func (l *localItem) Unlock() {
	l.L.Lock()
	defer l.L.Unlock()
	if !l.locked {
		logrus.Panicf("unlock unlocked lock %s", l.key)
	}
	l.locked = false
	if l.waiting > 0 {
		l.Signal()
	} else {
		l.lastTime = time.Now()
	}
}
