package rpc

import (
  "net/http"
  "regexp"
  "strings"
  "time"

  "github.com/tencent-go/pkg/ctxx"
  "github.com/tencent-go/pkg/errx"
  "github.com/sirupsen/logrus"
  clientv3 "go.etcd.io/etcd/client/v3"
)

type Method[I, O any] interface {
  WithTimeout(d time.Duration) Method[I, O]
  WithPort(p int) Method[I, O]
  WithEtcd(c *clientv3.Client) Method[I, O]
  WithServiceName(serviceName string) Method[I, O]
  WithDescription(description string) Method[I, O]
  Call(ctx ctxx.Context, cmd I) (*O, errx.Error)
  GetURL() (string, errx.Error)
  Handle(handler Handler[I, O])
}

func NewMethod[I, O any](path string) Method[I, O] {
  path = strings.Trim(path, "/")
  path = strings.TrimSpace(path)
  if !pathPattern.MatchString(path) {
    logrus.Panicf("invalid getOptions path: %s", path)
  }
  return &method[I, O]{options: options{path: path}}
}

type Context interface {
  ctxx.Context
  HttpRequest() *http.Request
  HttpWriter() http.ResponseWriter
}

type Handler[I, O any] func(ctx Context, params I) (*O, errx.Error)

type options struct {
  serviceName string
  path        string
  port        int
  timeout     time.Duration
  etcd        *clientv3.Client
  description string
}

type method[I, O any] struct {
  options
}

func (s *method[I, O]) Handle(handler Handler[I, O]) {
  serve(s.options, handler)
}

func (s *method[I, O]) WithTimeout(d time.Duration) Method[I, O] {
  o := s.options
  o.timeout = d
  return &method[I, O]{options: o}
}

func (s *method[I, O]) WithPort(p int) Method[I, O] {
  o := s.options
  o.port = p
  return &method[I, O]{options: o}
}

func (s *method[I, O]) WithEtcd(c *clientv3.Client) Method[I, O] {
  o := s.options
  o.etcd = c
  return &method[I, O]{options: o}
}

func (s *method[I, O]) WithServiceName(serviceName string) Method[I, O] {
  o := s.options
  o.serviceName = serviceName
  return &method[I, O]{options: o}
}

func (s *method[I, O]) WithDescription(description string) Method[I, O] {
  o := s.options
  o.description = description
  return &method[I, O]{options: o}
}

func (s *method[I, O]) Call(ctx ctxx.Context, cmd I) (*O, errx.Error) {
  return call[I, O](s.options, ctx, cmd)
}

func (s *method[I, O]) GetURL() (string, errx.Error) {
  return getURL(s.options)
}

var (
  pathPattern = regexp.MustCompile(`^[a-zA-Z0-9-_/{}]*$`)
)

const (
  defaultServerTimeout = 10 * time.Second
  defaultClientTimeout = 8 * time.Second
)

const etcdPrefix = "/rpc-services"
