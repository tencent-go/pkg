package keylocker

import (
	"github.com/tencent-go/pkg/etcdx"
	"github.com/tencent-go/pkg/shutdown"
	"context"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/client/v3/concurrency"
	"sync"
	"time"
)

type Locker interface {
	sync.Locker
	TryLock() bool
}

func Etcd(key string) Locker {
	val, _ := etcdLockers.LoadOrStore(key, sync.OnceValue(func() *etcdItem {
		return &etcdItem{
			local: Local(key).(*localItem),
			etcd:  concurrency.NewMutex(getSession(), key),
		}
	}))
	get := val.(func() *etcdItem)
	return get()
}

var etcdLockers sync.Map

type etcdItem struct {
	local *localItem
	etcd  *concurrency.Mutex
	count int
}

func (e *etcdItem) Lock() {
	// 先拿到本地鎖
	e.local.L.Lock()
	defer e.local.L.Unlock()
	for e.local.locked {
		e.local.waiting++
		e.local.Wait()
		e.local.waiting--
	}
	e.local.locked = true
	if e.count == 0 {
		ctx := context.Background()
		err := e.etcd.Lock(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logrus.Fatalf("failed to lock etcd mutex %s: %v", e.local.key, err)
		}
	}
	e.count++
}

func (e *etcdItem) TryLock() bool {
	e.local.L.Lock()
	defer e.local.L.Unlock()
	if e.local.locked {
		return false
	}
	if e.count == 0 {
		ctx := context.Background()
		err := e.etcd.TryLock(ctx)
		if err != nil {
			return false
		}
	}
	e.local.locked = true
	e.count++
	return true
}

func (e *etcdItem) Unlock() {
	e.local.L.Lock()
	defer e.local.L.Unlock()
	if !e.local.locked {
		logrus.Panicf("unlock unlocked lock %s", e.local.key)
	}
	e.local.locked = false
	if e.local.waiting > 0 {
		e.local.Signal()
	} else {
		e.local.lastTime = time.Now()
	}
	e.count--
	if e.count < 0 {
		logrus.Panicf("unlock unlocked lock %s", e.local.key)
	}
	if e.count == 0 {
		err := e.etcd.Unlock(context.Background())
		if err != nil {
			logrus.WithError(err).WithField("key", e.local.key).Error("failed to unlock etcd mutex")
		}
	}
}

var getSession = sync.OnceValue(func() *concurrency.Session {
	s, err := concurrency.NewSession(etcdx.DefaultClient(), concurrency.WithTTL(10))
	if err != nil {
		logrus.WithError(err).Fatal("failed to create etcd session")
	}
	shutdown.OnShutdown(func(ctx context.Context) error {
		return s.Close()
	}, true)
	return s
})
