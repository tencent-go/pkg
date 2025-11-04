package etcdx

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/tencent-go/pkg/env"
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/shutdown"
	"github.com/tencent-go/pkg/util"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// DefaultClient returns the default etcd client
func DefaultClient() *clientv3.Client {
	return defaultClient()
}

type Config struct {
	Endpoints []string `env:"ETCD_ENDPOINTS"`
}

var ConfigReaderBuilder = env.NewReaderBuilder[Config]()

var configReader = ConfigReaderBuilder.Build()

var defaultClient = sync.OnceValue(func() *clientv3.Client {
	c := configReader.Read()
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,     // `etcd` 服务的地址
		DialTimeout: 5 * time.Second, // 连接超时时间
	})
	if err != nil {
		logrus.WithError(err).WithField("endpoints", strings.Join(c.Endpoints, ",")).Panic("connect etcd failed")
	}
	shutdown.OnShutdown(func(ctx context.Context) error {
		defer logrus.Infoln("etcd connection closed")
		return cli.Close()
	}, true)
	logrus.Infoln("etcd connected")
	return cli
})

// Deprecated: use NewClientLeaseIDTracker instead
func GetClientUniqueLease(cli *clientv3.Client) *clientv3.LeaseGrantResponse {
	res, _ := leaseMap.LoadOrLazyStore(cli, func() *clientv3.LeaseGrantResponse {
		ctx, cancel := context.WithCancel(context.Background())
		lease, err := cli.Grant(ctx, 5)
		if err != nil {
			logrus.WithError(err).Panic("etcd grant lease failed")
		}
		ch, kaErr := cli.KeepAlive(ctx, lease.ID)
		if kaErr != nil {
			logrus.WithError(kaErr).Panic("etcd keepalive failed")
		}
		go func() {
			for {
				select {
				case ka, ok := <-ch:
					if !ok || ka == nil {
						logrus.Panic("KeepAlive channel closed")
					}
				case <-ctx.Done():
					logrus.Info("etcd keepalive closed")
					return
				}
			}
		}()
		shutdown.OnShutdown(func(ctx context.Context) error {
			cancel()
			_, err = cli.Revoke(ctx, lease.ID)
			return err
		}, true)
		return lease
	})
	return res
}

func NewClientLeaseIDTracker(cli *clientv3.Client) LeaseIdTracker {
	manager, _ := leaseIdTrackers.LoadOrLazyStore(cli, func() *leaseIdTrackerManager {
		return &leaseIdTrackerManager{cli: cli}
	})
	return &leaseIdTracker{m: manager}
}

type LeaseIdTracker interface {
	Track() chan clientv3.LeaseID
	Close()
}

type leaseIdTrackerManager struct {
	cli     *clientv3.Client
	list    map[chan clientv3.LeaseID]any
	mu      sync.RWMutex
	running bool
	current clientv3.LeaseID
	cancel  context.CancelFunc
}

func (lm *leaseIdTrackerManager) newTrack() chan clientv3.LeaseID {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.list == nil {
		lm.list = make(map[chan clientv3.LeaseID]any)
	}
	ch := make(chan clientv3.LeaseID)
	lm.list[ch] = nil
	if !lm.running {
		lm.start()
	}

	if lm.current != 0 {
		go func(id clientv3.LeaseID) {
			ch <- id
		}(lm.current)

	}
	return ch
}

func (lm *leaseIdTrackerManager) close(ch chan clientv3.LeaseID) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	delete(lm.list, ch)
	close(ch)
	if len(lm.list) == 0 && lm.running && lm.cancel != nil {
		lm.running = false
		lm.cancel()
	}
}

func (lm *leaseIdTrackerManager) start() {
	if lm.running {
		return
	}
	lm.running = true
	ctx, cancel := context.WithCancel(context.Background())
	lm.cancel = cancel

	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			id, disconnectCh, err := createAndKeepAliveLease(ctx, lm.cli)
			if err != nil {
				logrus.WithError(err).Error("etcd create and keep alive lease failed, retry in 5 seconds")
				time.Sleep(5 * time.Second)
				continue
			}
			lm.mu.RLock()
			lm.current = id
			if !lm.running {
				lm.mu.RUnlock()
				return
			}
			for ch := range lm.list {
				select {
				case ch <- id:
				default:
				}
			}
			lm.mu.RUnlock()
			select {
			case <-disconnectCh:
				continue
			case <-ctx.Done():
				return
			}
		}
	}()
}

func createAndKeepAliveLease(ctx context.Context, cli *clientv3.Client) (clientv3.LeaseID, chan any, errx.Error) {
	leaseResp, e := cli.Grant(ctx, 10)
	if e != nil {
		return 0, nil, errx.Wrap(e).Err()
	}
	kaCh, e := cli.KeepAlive(ctx, leaseResp.ID)
	if e != nil {
		return 0, nil, errx.Wrap(e).Err()
	}
	disconnectCh := make(chan any)
	go func() {
		defer close(disconnectCh)
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-kaCh:
				if !ok {
					return
				}
			}
		}
	}()
	return leaseResp.ID, disconnectCh, nil
}

type leaseIdTracker struct {
	m  *leaseIdTrackerManager
	ch chan clientv3.LeaseID
}

func (l *leaseIdTracker) Track() chan clientv3.LeaseID {
	if l.ch == nil {
		l.ch = l.m.newTrack()
	}
	return l.ch
}

func (l *leaseIdTracker) Close() {
	if l.ch != nil {
		l.m.close(l.ch)
		l.ch = nil
	}
}

var leaseIdTrackers = util.LazyMap[*clientv3.Client, *leaseIdTrackerManager]{}

var leaseMap = util.LazyMap[*clientv3.Client, *clientv3.LeaseGrantResponse]{}
