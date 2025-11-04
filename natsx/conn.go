package natsx

import (
	"context"
	"sync"
	"time"

	"github.com/tencent-go/pkg/env"

	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/shutdown"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Addresses string `env:"NATS_ADDRESSES" example:"nats://localhost:4222,nats://localhost:4223"`
}

var ConfigReaderBuilder = env.NewReaderBuilder[Config]()

var configReader = ConfigReaderBuilder.Build()

func GetDefaultConn() *nats.Conn {
	return getDefaultConn()
}

var getDefaultConn = sync.OnceValue(func() *nats.Conn {
	conn, err := ConnectWithDefaultOptions(configReader.Read().Addresses)
	if err != nil {
		logrus.WithError(err).Panic("connect nats failed")
		return nil
	}
	shutdown.OnShutdown(func(ctx context.Context) error {
		conn.Close()
		logrus.Info("nats connection closed")
		return nil
	}, true)
	logrus.Info("nats connected")
	return conn
})

func ConnectWithDefaultOptions(addresses string, options ...nats.Option) (*nats.Conn, errx.Error) {
	options = append([]nats.Option{
		func(options *nats.Options) error {
			options.Timeout = 30 * time.Second
			options.PingInterval = 3 * time.Second
			options.ReconnectWait = 30 * time.Second
			options.MaxPingsOut = 3
			options.AllowReconnect = true
			return nil
		},
	}, options...)
	c, err := nats.Connect(addresses, options...)
	if err != nil {
		return nil, errx.Wrap(err).Err()
	}
	return c, nil
}
