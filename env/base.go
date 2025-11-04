package env

import (
	"github.com/tencent-go/pkg/types"
)

var (
	BaseConfigReaderBuilder = NewReaderBuilder[Base]()
)

type Base struct {
	Env          Env      `env:"ENV" default:"prod"`
	LogLevel     LogLevel `env:"LOG_LEVEL" default:"info"`
	AppName      string   `env:"APP_NAME,omitempty"`
	PodName      string   `env:"POD_NAME" default:"pod"`
	Initializing bool     `env:"INITIALIZING" default:"false"`
}

type Env string

func (E Env) Enum() types.Enum {
	return types.RegisterEnum(DEV, PROD, TEST, STAG)
}

const (
	DEV  Env = "dev"
	PROD Env = "prod"
	TEST Env = "test"
	STAG Env = "stag"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

func (l LogLevel) Enum() types.Enum {
	return types.RegisterEnum(LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError)
}
