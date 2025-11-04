package keylocker

import (
	"sync"
	"time"
)

func SetCleanupInterval(interval time.Duration) {
	cleanupInterval = interval
}

func SetCleanupTime(expiration time.Duration) {
	cleanupTime = expiration
}

var cleanupInterval = time.Minute
var cleanupTime = time.Minute

var startAutoClear = sync.OnceFunc(func() {
	go func() {
		for {
			time.Sleep(cleanupInterval)
			localLockers.Range(func(key, value interface{}) bool {
				get := value.(func() *localItem)
				item := get()
				if item.waiting == 0 && !item.locked && time.Since(item.lastTime) > cleanupTime {
					etcdLockers.Delete(key)
					localLockers.Delete(key)
				}
				return true
			})
		}
	}()
})
