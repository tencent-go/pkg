package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/etcdx"
	"github.com/tencent-go/pkg/shutdown"
	"github.com/tencent-go/pkg/types"
	"github.com/tencent-go/pkg/util"
	"github.com/tencent-go/pkg/validation"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func serve[I, O any](o options, handler Handler[I, O]) {
	config := configReader.Read()
	baseConfig := baseConfigReader.Read()
	serviceName := o.serviceName
	if serviceName == "" {
		serviceName = config.RpcServiceName
		if serviceName == "" {
			logrus.Panic("no service name set")
		}
	}
	if o.path == "" {
		logrus.Panic("no path set")
	}
	if o.port == 0 {
		o.port = config.RpcServerPort
	}
	if o.timeout > 0 {
		o.timeout = o.timeout + time.Second*2
	}
	if o.timeout == 0 {
		o.timeout = defaultServerTimeout
	}
	if o.etcd == nil {
		o.etcd = defaultEtcd()
	}
	path := fmt.Sprintf("/%s/%s", serviceName, o.path)
	key := fmt.Sprintf("%s%s/%s", etcdPrefix, path, baseConfig.PodName)
	value := getMethodEtcdValue(config.Namespace, o.port)
	srv, _ := httpServers.LoadOrLazyStore(o.port, func() *httpServer {
		return newHttpServer(o.port)
	})
	if err := srv.addMethodHandler(path, newHttpHandler(handler, o.timeout)); err != nil {
		panic(err)
	}
	//註冊
	go func() {
		for lid := range etcdx.NewClientLeaseIDTracker(o.etcd).Track() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			if _, err := o.etcd.Put(ctx, key, value, clientv3.WithLease(lid)); err != nil {
				logrus.WithError(err).Panicf("register service method to etcd %s failed", key)
			}
			cancel()
		}
	}()
}

func getMethodEtcdValue(namespace string, port int) string {
	return fmt.Sprintf("%s,%d", namespace, port)
}

func newHttpServer(port int) *httpServer {
	srv := &httpServer{}
	hSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: h2c.NewHandler(srv, &http2.Server{}),
	}
	go func() {
		if e := hSrv.ListenAndServe(); e != nil {
			if errors.Is(e, http.ErrServerClosed) {
				logrus.Infof("rpc http server stopped on port %d", port)
				return
			}
			logrus.WithError(e).Panicf("rpc http server failed to start on port %d", port)
		}
	}()
	shutdown.OnShutdown(hSrv.Shutdown, true)
	logrus.Infof("http server started on port %d", port)
	return srv
}

type httpServer struct {
	handlers map[string]http.HandlerFunc
	mu       sync.RWMutex
}

func (s *httpServer) addMethodHandler(path string, handler http.HandlerFunc) errx.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handlers == nil {
		s.handlers = make(map[string]http.HandlerFunc)
	}
	if _, ok := s.handlers[path]; ok {
		return errx.Newf("handler for path %s already exists", path)
	}
	s.handlers[path] = handler
	return nil
}

func (s *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path
	if serviceName := r.Header.Get("rpc-service-name"); serviceName != "" {
		key = serviceName + "/" + key
	}
	if h, ok := s.handlers[key]; ok {
		h(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

var httpServers = util.LazyMap[int, *httpServer]{}

func newHttpHandler[I, O any](handler Handler[I, O], timeout time.Duration) http.HandlerFunc {
	validate, shouldValidate, _ := validation.GetOrCreateValidator(reflect.TypeOf(new(I)))
	return func(w http.ResponseWriter, req *http.Request) {
		log := logrus.WithField("path", req.URL.Path)
		w.Header().Set("Content-Type", "application/msgpack")
		ctx := ctxx.WithMetadata(req.Context(), readMetadataFromHeaders(req.Header))
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = ctxx.WithTimeout(ctx, timeout)
			defer cancel()
		}
		log = log.WithContext(ctx)
		log.Debug("received rpc request")
		input, err := ReadRequestBody[I](req)
		if err != nil {
			log.WithError(err).Error("read data failed")
			WriteError(w, err)
			return
		}
		if shouldValidate {
			if err = validate(input); err != nil {
				log.WithError(err).Error("validate failed")
				WriteError(w, err)
				return
			}
		}
		start := time.Now()
		cw := &contextWrapper{
			Context: ctx,
			r:       req,
			w:       w,
		}
		output, err := handler(cw, *input)
		log = log.WithField("duration", time.Since(start).String())
		if logrus.GetLevel() >= logrus.DebugLevel || err != nil {
			log = log.WithField("response", output).WithField("request", input)
		}

		if err != nil {
			WriteError(w, err)
			if err.Type() == errx.TypeInternal {
				log.WithError(err).Error("handle rpc request failed")
			} else {
				log.WithError(err).Warn("handle rpc request failed")
			}
		} else {
			log.Info("handle rpc request successful")
			if err = WriteSuccess(w, output); err != nil {
				log.WithError(err).Error("write response failed")
			}
		}
	}
}

type contextWrapper struct {
	ctxx.Context
	r *http.Request
	w http.ResponseWriter
}

func (c *contextWrapper) HttpRequest() *http.Request {
	return c.r
}

func (c *contextWrapper) HttpWriter() http.ResponseWriter {
	return c.w
}

func readMetadataFromHeaders(headers http.Header) ctxx.Metadata {
	tid, _ := types.NewIDFromString(headers.Get("rpc-trace-id"))
	return ctxx.Metadata{
		TraceID:  tid,
		Operator: headers.Get("rpc-operator"),
		Caller:   headers.Get("rpc-caller"),
		Locale:   types.Locale(headers.Get("rpc-locale")),
	}
}
