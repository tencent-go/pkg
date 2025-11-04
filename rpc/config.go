package rpc

import (
	"github.com/tencent-go/pkg/env"
	"github.com/tencent-go/pkg/etcdx"
)

type Config struct {
	ServiceDomainSuffix string `env:"SERVICE_DOMAIN_SUFFIX" default:"cluster.local"`
	RpcServiceName      string `env:"RPC_SERVICE_NAME" default:"localhost"`
	Namespace           string `env:"NAMESPACE" default:"localhost"`
	RpcServerPort       int    `env:"RPC_SERVER_PORT" default:"28000"`
}

var (
	etcdConfigReader = etcdx.ConfigReaderBuilder.WithPrefix("RPC").WithAllFieldsRequired(false).Build()
	configReader     = env.NewReaderBuilder[Config]().Build()
	baseConfigReader = env.BaseConfigReaderBuilder.Build()
)
