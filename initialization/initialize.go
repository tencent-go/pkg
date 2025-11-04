package initialization

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/env"
	"github.com/tencent-go/pkg/etcdx"
	"github.com/tencent-go/pkg/keylocker"
	"github.com/tencent-go/pkg/types"
	"github.com/sirupsen/logrus"
)

var baseConfigReader = env.BaseConfigReaderBuilder.Build()

func Initialize() {
	c := baseConfigReader.Read()
	applyLogSetting(convertLevel(c.LogLevel))
	types.SetIDNodeByString(c.PodName)
}

type LogLevel string

func convertLevel(l env.LogLevel) logrus.Level {
	switch l {
	case env.LogLevelDebug:
		return logrus.DebugLevel
	case env.LogLevelInfo:
		return logrus.InfoLevel
	case env.LogLevelWarn:
		return logrus.WarnLevel
	case env.LogLevelError:
		return logrus.ErrorLevel
	default:
		return 0
	}
}

var shouldInitialize = os.Getenv("INITIALIZING") == "true"

// Register 如有tag 則會在 etcd 中建立 /initializing/tag 的 key 來防止重複初始化
func Register(fn func(ctx ctxx.Context), tag ...string) {
	if !shouldInitialize {
		return
	}
	ctx := ctxx.WithMetadata(context.Background(), ctxx.Metadata{Operator: "initiator"})
	var key string
	if len(tag) > 0 {
		key = path.Join("/initialize/", path.Join(tag...))
	}
	defer func() {
		if r := recover(); r != nil {
			logrus.Fatalf("panic on init %s: %v", key, r)
			return
		}
		if key != "" {
			v := time.Now().Format(time.RFC3339)
			_, err := etcdx.DefaultClient().Put(context.Background(), key, v)
			if err != nil {
				logrus.Errorf("failed to put etcd key %s: %v", key, err)
			}
		}
	}()
	var locker keylocker.Locker
	if key != "" {
		locker = keylocker.Etcd(key)
		if !locker.TryLock() {
			return
		}
		defer locker.Unlock()
		res, err := etcdx.DefaultClient().Get(ctx, key)
		if err != nil {
			logrus.Fatalf("failed to get etcd key %s: %v", key, err)
		}
		if res.Count > 0 {
			logrus.Infof("skip init %s", key)
			return
		}
	}
	fn(ctx)
}
