package rpc

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	path2 "path"
	"reflect"
	"sync"

	"github.com/tencent-go/pkg/errx"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v5"
)

type Group struct {
	Description string
	Path        string
	Routes      []Route
}

type Route interface {
	http.Handler
	InputType() reflect.Type
	OutputType() reflect.Type
	Path() string
	Description() string
}

func ProxyRoute[I, O any](path string, met Method[I, O], description ...string) Route {
	var d string
	if m, ok := met.(*method[I, O]); ok {
		d = m.description
		if path == "" {
			path = m.path
		}
	}
	if len(description) > 0 {
		d = description[0]
	}
	return &route{
		path:        path,
		inputType:   reflect.TypeOf(new(I)).Elem(),
		outputType:  reflect.TypeOf(new(O)).Elem(),
		description: d,
		handler: func(w http.ResponseWriter, r *http.Request) {
			u, err := met.GetURL()
			if err != nil {
				WriteError(w, err)
				return
			}
			req, e := http.NewRequestWithContext(r.Context(), "POST", u, r.Body)
			if e != nil {
				WriteError(w, errx.Wrap(e).Err())
				return
			}
			res, e := httpClient.Do(req)
			if e != nil {
				WriteError(w, errx.Wrap(e).Err())
				return
			}
			defer func() {
				_ = res.Body.Close()
			}()
			for k, vv := range res.Header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(res.StatusCode)
			if res.StatusCode >= 500 {
				w.Header().Del("Content-Length")
				WriteError(w, errx.Internal.Err())
				return
			}
			if _, e = io.Copy(w, res.Body); e != nil {
				logrus.WithError(e).Error("failed to write response")
			}
		},
	}
}

func HandleRoute[I, O any](path string, handler Handler[I, O], description ...string) Route {
	var d string
	if len(description) > 0 {
		d = description[0]
	}
	return &route{
		path:        path,
		inputType:   reflect.TypeOf(new(I)).Elem(),
		outputType:  reflect.TypeOf(new(O)).Elem(),
		description: d,
		handler:     newHttpHandler(handler, 0),
	}
}

type Interceptor func(w http.ResponseWriter, r *http.Request, route Route, group Group) bool

func NewGateway(groups []Group, interceptors ...Interceptor) Gateway {
	g := &gateway{
		groups:       groups,
		interceptors: interceptors,
	}
	for _, group := range groups {
		for _, r := range group.Routes {
			path := path2.Join("/", group.Path, r.Path())
			_, exists := g.routeByPath.LoadOrStore(path, &pathRoute{route: r, group: group})
			if exists {
				panic("duplicate route " + path)
			}
		}
	}
	return g
}

type Gateway interface {
	http.Handler
	Groups() []Group
}

type pathRoute struct {
	route Route
	group Group
}

type gateway struct {
	groups       []Group
	interceptors []Interceptor
	routeByPath  sync.Map
}

func (g *gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	item, ok := g.routeByPath.Load(r.URL.Path)
	if !ok {
		WriteError(w, errx.NotFound.WithMsgf("path [%s] not found", r.URL.Path).Err())
		return
	}
	matchedRoute := item.(*pathRoute)
	for _, in := range g.interceptors {
		if !in(w, r, matchedRoute.route, matchedRoute.group) {
			return
		}
	}
	matchedRoute.route.ServeHTTP(w, r)
}

func (g *gateway) Groups() []Group {
	return g.groups
}

type route struct {
	inputType   reflect.Type
	outputType  reflect.Type
	path        string
	description string
	handler     http.HandlerFunc
}

func (r *route) ServeHTTP(writer http.ResponseWriter, r2 *http.Request) {
	r.handler(writer, r2)
}

func (r *route) InputType() reflect.Type {
	return r.inputType
}

func (r *route) OutputType() reflect.Type {
	return r.outputType
}

func (r *route) Path() string {
	return r.path
}

func (r *route) Description() string {
	return r.description
}

func NewJsonTranscoder(dst string) http.HandlerFunc {
	client := &http.Client{}
	writeError := func(w http.ResponseWriter, err errx.Error) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	return func(w http.ResponseWriter, r *http.Request) {

		u, e := url.JoinPath(dst, r.URL.Path)
		if e != nil {
			writeError(w, errx.Wrap(e).Err())
			return
		}
		var reqBody io.Reader
		if r.ContentLength != 0 {
			var v any
			dec := json.NewDecoder(r.Body)
			dec.UseNumber()
			if e = dec.Decode(&v); e != nil {
				writeError(w, errx.Wrap(e).Err())
				return
			}
			buf := &bytes.Buffer{}
			if e = msgpack.NewEncoder(buf).Encode(v); e != nil {
				writeError(w, errx.Wrap(e).Err())
				return
			}
			reqBody = buf
		}
		req, e := http.NewRequestWithContext(r.Context(), "POST", u, reqBody)
		if e != nil {
			writeError(w, errx.Wrap(e).Err())
			return
		}
		for k, vv := range r.Header {
			for _, v := range vv {
				req.Header.Add(k, v)
			}
		}
		resp, e := client.Do(req)
		if e != nil {
			writeError(w, errx.Wrap(e).Err())
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.Header().Del("Content-Length")
		var resBody []byte
		if resp.ContentLength != 0 {
			var v any
			if e = msgpack.NewDecoder(resp.Body).Decode(&v); e != nil {
				WriteError(w, errx.Wrap(e).Err())
				return
			}
			resBody, e = json.Marshal(v)
			if e != nil {
				writeError(w, errx.Wrap(e).Err())
				return
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(resBody)
	}
}
