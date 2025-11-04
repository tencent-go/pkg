package redisx

import (
	"github.com/tencent-go/pkg/env"
	"github.com/tencent-go/pkg/shutdown"
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"sync"
)

type Config struct {
	Address  string `env:"REDIS_ADDRESS" example:"127.0.0.1:6379"`
	Password string `env:"REDIS_PASSWORD,omitempty" example:"123456"`
}

var ConfigReaderBuilder = env.NewReaderBuilder[Config]()

var configReader = ConfigReaderBuilder.Build()

var defaultConfig = Config{
	Address:  "[host]:[port]",
	Password: "[password]",
}

func GetDefaultClient() *redis.ClusterClient {
	return getClient()
}

var getClient = sync.OnceValue(func() *redis.ClusterClient {
	cfg := configReader.Read()
	cli := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    []string{cfg.Address},
		Password: cfg.Password,
	})
	ctx := context.Background()
	_, err := cli.Ping(ctx).Result()
	if err != nil {
		logrus.WithError(err).Fatal("redis connection failed")
	}
	shutdown.OnShutdown(func(ctx context.Context) error {
		return cli.Close()
	}, true)
	logrus.Info("redis connected")
	return cli
})
