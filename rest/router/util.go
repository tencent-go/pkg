package router

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/tencent-go/pkg/types"
	"github.com/tencent-go/pkg/util"
	"github.com/tencent-go/pkg/validation"
	"github.com/sirupsen/logrus"

	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/rest/api"
)

func language2Locale(value string) types.Locale {
	if value == "" {
		return ""
	}

	type item struct {
		l types.Locale
		w float64
	}

	var items []item
	parts := strings.Split(value, ",")

	for _, part := range parts {
		langAndQuality := strings.Split(strings.TrimSpace(part), ";")
		if len(langAndQuality) == 0 {
			continue
		}

		lang := types.Locale(langAndQuality[0])
		if ok := lang.Enum().Contains(lang); !ok {
			continue
		}

		weight := 1.0
		if len(langAndQuality) > 1 && strings.HasPrefix(langAndQuality[1], "q=") {
			if _, e := fmt.Sscanf(langAndQuality[1], "q=%f", &weight); e != nil {
				continue
			}
		}

		items = append(items, item{l: lang, w: weight})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].w > items[j].w
	})

	if len(items) > 0 {
		return items[0].l
	}

	return ""
}

var apiHeaderSerializer = sync.OnceValue(func() util.HttpSerializer[api.HeaderParams] {
	res, _ := util.NewHttpSerializer[api.HeaderParams](util.TagHeader)
	return res
})

func GetHeaderParams(ctx Context) (*api.HeaderParams, errx.Error) {
	h := &api.HeaderParams{}
	err := apiHeaderSerializer().Deserialize(&util.HttpData{
		Header: ctx.Request().Header,
	}, h)
	if err != nil {
		return nil, err
	}
	if h.Locale != "" {
		if ok := h.Locale.Enum().Contains(h.Locale); !ok {
			h.Locale = ""
		}
	}
	if h.Locale == "" {
		h.Locale = ctx.GetLocale()
	}
	if h.RealIP == "" {
		if forwarded := ctx.Request().Header.Get("X-Forwarded-For"); forwarded != "" {
			h.RealIP = strings.Split(forwarded, ",")[0]
		}
		if h.RealIP == "" {
			h.RealIP = ctx.Request().RemoteAddr
		}
	}
	return h, nil
}

func ReadBodyReusable(req *http.Request) ([]byte, errx.Error) {
	if req.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, errx.Wrap(err).Err()
	}
	if e := req.Body.Close(); e != nil {
		logrus.WithError(e).Error("Failed to close body")
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

type endpointHandler struct {
	endpoint api.Endpoint
	handler  HandlerFunc
}

func (e *endpointHandler) Endpoint() api.Endpoint {
	return e.endpoint
}

func (e *endpointHandler) Handle(ctx Context) {
	e.handler(ctx)
}

func NewEndpointHandler[I, O any](endpoint api.EndpointBuilder[I, O], handler func(ctx Context, params I) (*O, errx.Error)) EndpointHandler {
	inputSerializer := util.LazyMap[util.StructTag, util.HttpSerializer[I]]{}
	var validate func(any) errx.Error
	getInputSerializer := func(contentType api.ContentType) util.HttpSerializer[I] {
		var t util.StructTag
		switch contentType {
		case api.ContentTypeApplicationJson:
			t = util.TagJson
		case api.ContentTypeApplicationFormUrlencoded:
			t = util.TagForm
		default:
			return nil
		}
		s, _ := inputSerializer.LoadOrLazyStore(t, func() util.HttpSerializer[I] {
			s, _ := util.NewHttpSerializer[I](t, util.TagHeader, util.TagPath, util.TagQuery)
			return s
		})
		validate, _, _ = validation.GetOrCreateValidator(endpoint.InputType(), validation.WithLabelTags("query", "path", string(t)))
		return s
	}

	h := func(ctx Context) {
		var input I
		if t := ctx.Endpoint().InputType(); t != nil && t.Kind() == reflect.Struct {
			if s := getInputSerializer(ctx.RequestContentType()); s != nil {
				body, err := ReadBodyReusable(ctx.Request())
				if err != nil {
					ctx.State().Error = err
					return
				}
				data := &util.HttpData{
					Header: ctx.Request().Header,
					Query:  ctx.Request().URL.Query(),
					Path:   ctx.PathParams(),
					Body:   body,
				}
				if err = s.Deserialize(data, &input); err != nil {
					ctx.State().Error = err
					return
				}
				if validate != nil {
					if err = validate(input); err != nil {
						ctx.State().Error = err
						return
					}
				}
			}
		}
		res, err := handler(ctx, input)
		if err != nil {
			ctx.State().Error = err
			return
		}
		if t := ctx.Endpoint().OutputType(); t != nil {
			if ctx.ResponseContentType() == api.ContentTypeApplicationJson {
				ctx.State().Data, ctx.State().Error = util.Json().Marshal(res)
			}
			if t.Kind() == reflect.Struct && ctx.ResponseContentType() == api.ContentTypeApplicationFormUrlencoded {
				ctx.State().Data, ctx.State().Error = util.Form().Marshal(res)
			}
		}
	}
	return &endpointHandler{
		endpoint: endpoint,
		handler:  h,
	}
}

func RegisterEndpointHandler[I, O any](router Router, endpoint api.EndpointBuilder[I, O], handler func(ctx Context, params I) (*O, errx.Error)) {
	h := NewEndpointHandler(endpoint, handler)
	router.HandleEndpoints(h)
}

func checkPathParamKeys(routes []api.Route) errx.Error {
	if len(routes) == 0 {
		return nil
	}
	for _, r := range routes {
		if len(r.PathChain()) == 0 {
			continue
		}
		matches := util.PlaceholderRegex.FindAllStringSubmatch(r.Path(), -1)
		if len(matches) == 0 {
			continue
		}
		t := r.Endpoint().InputType()
		if t == nil || t.Kind() != reflect.Struct {
			return errx.New("invalid input type")
		}
		for _, m := range matches {
			if !isPathFieldExists(t, m[1]) {
				return errx.Newf("invalid path field: %s", m[1])
			}
		}
	}
	return nil
}

func isPathFieldExists(typ reflect.Type, field string) bool {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue
		}
		if f.Anonymous {
			if isPathFieldExists(f.Type, field) {
				return true
			}
		}
		if !f.IsExported() {
			continue
		}
		tag, ok := f.Tag.Lookup("path")
		if !ok {
			continue
		}
		if idx := strings.Index(tag, ","); idx != -1 {
			tag = tag[:idx]
		}
		if tag == field {
			return true
		}
	}
	return false
}
