package ctxx

import (
	"context"
	"time"

	"github.com/tencent-go/pkg/types"
)

type Context interface {
	context.Context
	GetTraceID() types.ID
	GetCaller() string
	GetLocale() types.Locale
	GetOperator() string
}

type wrapper struct {
	context.Context
	*Metadata
}

func GetMetadata(ctx Context) *Metadata {
	if ctx == nil {
		m := &Metadata{}
		m.FillDefaults()
		return m
	}
	if m, ok := ctx.(*wrapper); ok {
		return m.Metadata
	}
	return &Metadata{
		TraceID:  ctx.GetTraceID(),
		Operator: ctx.GetOperator(),
		Caller:   ctx.GetCaller(),
		Locale:   ctx.GetLocale(),
	}
}

func Background() Context {
	m := &Metadata{}
	m.FillDefaults()
	return &wrapper{Context: context.Background(), Metadata: m}
}

func WithMetadata(_ctx context.Context, metadata Metadata) Context {
	m := &metadata
	m.FillDefaults()
	return &wrapper{Context: _ctx, Metadata: m}
}

func WithContext(ctx context.Context) Context {
	var metadata *Metadata
	if c, ok := ctx.(Context); ok {
		metadata = GetMetadata(c)
	} else {
		metadata = &Metadata{}
		metadata.FillDefaults()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &wrapper{Context: ctx, Metadata: metadata}
}

func WithTimeout(parent Context, timeout time.Duration) (Context, context.CancelFunc) {
	var _ctx context.Context = parent
	var metadata *Metadata
	if parent != nil {
		if m, ok := parent.(*wrapper); ok {
			metadata = m.Metadata
		} else {
			metadata = &Metadata{
				TraceID:  parent.GetTraceID(),
				Operator: parent.GetOperator(),
				Caller:   parent.GetCaller(),
				Locale:   parent.GetLocale(),
			}
		}
	} else {
		metadata = &Metadata{}
		metadata.FillDefaults()
		_ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(_ctx, timeout)
	newCtx := &wrapper{Context: ctx, Metadata: metadata}
	return newCtx, cancel
}
