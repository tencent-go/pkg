package router

import (
	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/rest/api"
	"github.com/tencent-go/pkg/util"
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync"
)

type Context interface {
	ctxx.Context
	api.MatchedRoute
	Request() *http.Request
	ResponseWriter() ResponseWriter
	State() *State
	Next()
	Storage() util.Storage
}

type ResponseWriter interface {
	http.ResponseWriter
	http.Hijacker
	HeaderWritten() bool
	StatusCode() int
	BodyWritten() bool
}

type responseWriter struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
	bodyWritten   bool
}

func (r *responseWriter) HeaderWritten() bool {
	return r.headerWritten
}

func (r *responseWriter) StatusCode() int {
	return r.statusCode
}

func (r *responseWriter) BodyWritten() bool {
	return r.bodyWritten
}

func (r *responseWriter) Write(data []byte) (int, error) {
	if !r.headerWritten {
		r.statusCode = http.StatusOK
	}
	r.headerWritten = true
	r.bodyWritten = true
	return r.ResponseWriter.Write(data)
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.headerWritten = true
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("hijacker not supported")
}

type State struct {
	HttpStatus int
	Data       []byte
	Error      errx.Error
}

type context struct {
	ctxx.Context
	api.MatchedRoute
	request  *http.Request
	response *responseWriter
	state    *State
	storage  sync.Map
	next     func()
}

func (c *context) Request() *http.Request {
	return c.request
}

func (c *context) ResponseWriter() ResponseWriter {
	return c.response
}

func (c *context) State() *State {
	return c.state
}

func (c *context) Next() {
	if c.next != nil {
		c.next()
	}
}

func (c *context) Storage() util.Storage {
	return &c.storage
}
