package router

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"

	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/rest/api"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Router interface {
	http.Handler
	GetRoute(endpoint api.Endpoint) (api.Route, bool)
	GetRoutes() []api.Route
	AddNodes(nodes ...api.Node)
	UseRootMiddlewares(middlewares ...HandlerFunc)
	UsePathMiddlewares(method api.Method, path string, middlewares ...HandlerFunc)
	UseGroupMiddlewares(group api.Group, middlewares ...HandlerFunc)
	HandleEndpoint(endpoint api.Endpoint, handler HandlerFunc)
	HandleEndpoints(endpointHandlers ...EndpointHandler)
	HandleNotFound(handlerFunc http.HandlerFunc)
	Run(addr string) error
	RunH2C(addr string) error
}

type HandlerFunc func(ctx Context)

type EndpointHandler interface {
	Endpoint() api.Endpoint
	Handle(ctx Context)
}

func New() Router {
	return &router{
		rootGroup: api.NewGroup(),
	}
}

func NewWithDefaultMiddlewares() Router {
	r := New()
	r.UseRootMiddlewares(
		LoggerMiddleware(),
		JsonResponseWrapMiddleware(),
	)
	r.HandleNotFound(NotFoundHandler())
	return r
}

type router struct {
	rootGroup        api.GroupBuilder
	rootMiddlewares  []HandlerFunc
	pathMiddlewares  map[string][]HandlerFunc
	groupMiddlewares map[api.Node][]HandlerFunc
	endpointHandlers map[api.Endpoint]HandlerFunc
	notFoundHandler  http.HandlerFunc
}

func noHandler(ctx Context) {
	ctx.State().Error = errx.Newf("handler not found for %s %s", ctx.Endpoint().Method(), ctx.Path())
	ctx.State().HttpStatus = http.StatusServiceUnavailable
}

func (r *router) GetRoute(endpoint api.Endpoint) (api.Route, bool) {
	for _, route := range r.rootGroup.Routes() {
		if route.Endpoint() == endpoint {
			return route, true
		}
	}
	return nil, false
}

func (r *router) GetRoutes() []api.Route {
	return r.rootGroup.Routes()
}

func (r *router) AddNodes(nodes ...api.Node) {
	if len(nodes) == 0 {
		return
	}
	r.rootGroup = r.rootGroup.WithChildren(nodes...)
	for _, node := range nodes {
		if g, ok := node.(api.Group); ok {
			if err := checkPathParamKeys(g.Routes()); err != nil {
				logrus.WithError(err).Error("check path param keys failed")
			}
		}
	}
}

func (r *router) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	matchedRoute, ok := r.rootGroup.Match(api.Method(request.Method), request.URL.Path)
	if !ok {
		if r.notFoundHandler != nil {
			r.notFoundHandler.ServeHTTP(writer, request)
			return
		}
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	// 收集所有中间件
	middlewares := make([]HandlerFunc, 0, len(r.rootMiddlewares)+10)

	// 添加根中间件
	middlewares = append(middlewares, r.rootMiddlewares...)

	// 添加路径中间件
	if len(r.pathMiddlewares) > 0 {
		keys := []string{string(matchedRoute.Endpoint().Method())}
		for _, path := range matchedRoute.PathChain() {
			key := strings.Join(keys, "|")
			if pathMiddlewares, ok := r.pathMiddlewares[key]; ok {
				middlewares = append(middlewares, pathMiddlewares...)
			}
			keys = append(keys, path)
		}
	}

	// 添加节点中间件
	if len(r.groupMiddlewares) > 0 {
		for _, group := range matchedRoute.Ancestors() {
			if groupMiddlewares, ok := r.groupMiddlewares[group]; ok {
				middlewares = append(middlewares, groupMiddlewares...)
			}
		}
	}

	// 添加端点处理器
	handler, hasHandler := r.endpointHandlers[matchedRoute.Endpoint()]
	if hasHandler {
		middlewares = append(middlewares, handler)
	} else {
		middlewares = append(middlewares, noHandler)
	}

	// 创建上下文
	ctx := &context{
		Context: ctxx.WithMetadata(request.Context(), ctxx.Metadata{
			Locale: language2Locale(request.Header.Get("Accept-Language")),
		}),
		MatchedRoute: matchedRoute,
		request:      request,
		response:     &responseWriter{ResponseWriter: writer},
		state:        &State{},
		next:         nil,
	}

	// 构建中间件链
	if len(middlewares) > 0 {
		// 创建中间件执行函数
		var executeMiddleware func(index int)

		executeMiddleware = func(index int) {
			if index >= len(middlewares) {
				return // 所有中间件执行完毕
			}

			currentMiddleware := middlewares[index]

			// 设置下一个中间件的执行函数
			ctx.next = func() {
				executeMiddleware(index + 1)
			}

			// 执行当前中间件
			currentMiddleware(ctx)
		}

		// 开始执行第一个中间件
		executeMiddleware(0)
	}
}

func (r *router) UseRootMiddlewares(middlewares ...HandlerFunc) {
	r.rootMiddlewares = append(r.rootMiddlewares, middlewares...)
}

func (r *router) UsePathMiddlewares(method api.Method, path string, middlewares ...HandlerFunc) {
	if r.pathMiddlewares == nil {
		r.pathMiddlewares = make(map[string][]HandlerFunc)
	}
	key := strings.Join([]string{string(method), path}, "|")
	r.pathMiddlewares[key] = append(r.pathMiddlewares[key], middlewares...)
}

func (r *router) UseGroupMiddlewares(group api.Group, middlewares ...HandlerFunc) {
	if r.groupMiddlewares == nil {
		r.groupMiddlewares = make(map[api.Node][]HandlerFunc)
	}
	r.groupMiddlewares[group] = append(r.groupMiddlewares[group], middlewares...)
}

func (r *router) HandleEndpoint(endpoint api.Endpoint, handler HandlerFunc) {
	if r.endpointHandlers == nil {
		r.endpointHandlers = make(map[api.Endpoint]HandlerFunc)
	}
	r.endpointHandlers[endpoint] = handler
}

func (r *router) HandleEndpoints(endpointHandlers ...EndpointHandler) {
	if r.endpointHandlers == nil {
		r.endpointHandlers = make(map[api.Endpoint]HandlerFunc)
	}
	for _, item := range endpointHandlers {
		r.endpointHandlers[item.Endpoint()] = item.Handle
	}
}

func (r *router) HandleNotFound(handlerFunc http.HandlerFunc) {
	r.notFoundHandler = handlerFunc
}

func (r *router) Run(addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	return server.ListenAndServe()
}

func (r *router) RunH2C(addr string) error {
	httpServer := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(r, &http2.Server{}),
	}
	return httpServer.ListenAndServe()
}
