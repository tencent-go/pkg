package shutdown

import (
	"context"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type callback func(ctx context.Context) error

var (
	asyncStops []callback
	syncStops  []callback
	mu         sync.RWMutex
	timeout    = time.Second * 10
)

func Wait() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	logrus.Info("shutdown signal received")
	mu.RLock()
	defer mu.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	stopAsync(ctx)
	stopSync(ctx)
	logrus.Info("shutdown completed")
}

// OnShutdown 第二個參數為是否同步處理 如果是 則按先進後出順序處理
func OnShutdown(cb callback, args ...bool) {
	mu.Lock()
	defer mu.Unlock()
	s := false
	if len(args) > 0 {
		s = args[0]
	}
	if !s {
		asyncStops = append(asyncStops, cb)
	} else {
		syncStops = append(syncStops, cb)
	}
}

func SetTimeout(d time.Duration) {
	mu.Lock()
	defer mu.Unlock()
	timeout = d
}

func stopSync(ctx context.Context) {
	for i := len(syncStops) - 1; i >= 0; i-- {
		cb := syncStops[i]
		if err := cb(ctx); err != nil {
			logrus.WithError(err).Error("shutdown error")
		}
	}
}

func stopAsync(ctx context.Context) {
	wg := sync.WaitGroup{}
	for _, s := range asyncStops {
		wg.Add(1)
		go func(s callback) {
			defer wg.Done()
			if err := s(ctx); err != nil {
				logrus.WithError(err).Error("shutdown error")
			}
		}(s)
	}
	wg.Wait()
}

func Shutdown() {
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(syscall.SIGINT)
}
