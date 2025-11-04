package router

import (
	"github.com/tencent-go/pkg/rest/api"
	"github.com/tencent-go/pkg/util"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"net/http"
	"time"

	"github.com/tencent-go/pkg/errx"
	"github.com/sirupsen/logrus"
)

func LoggerMiddleware() HandlerFunc {
	return func(ctx Context) {
		t := time.Now()
		ctx.Next()
		req := ctx.Request()
		fields := logrus.Fields{
			"endpoint": fmt.Sprintf("%s %s", req.Method, req.URL.Path),
			"duration": time.Since(t).String(),
		}
		if q := req.URL.Query(); len(q) > 0 {
			fields["query"] = q
		}
		err := ctx.State().Error
		if logrus.GetLevel() > logrus.InfoLevel || err != nil {
			fields["requestHeader"] = req.Header
			if body, err := ReadBodyReusable(ctx.Request()); err == nil {
				fields["requestBody"] = string(body)
			}
			fields["responseHeader"] = ctx.ResponseWriter().Header()
			fields["responseBody"] = string(ctx.State().Data)
		}
		log := logrus.WithTime(t).WithContext(ctx).WithFields(fields)
		if err == nil {
			log.Info("handle request success")
		} else {
			if err.Type() == errx.TypeInternal {
				log.WithError(err).Error("handle request failed")
			} else {
				log.WithError(err).Warn("handle request failed")
			}
		}
	}
}

type ErrorDetails struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Type    errx.Type `json:"type"`
}

type JsonResponseWrapper struct {
	Success bool                `json:"success"`
	Data    jsoniter.RawMessage `json:"data,omitempty"`
	Error   *ErrorDetails       `json:"error,omitempty"`
}

func JsonResponseWrapMiddleware() HandlerFunc {
	return func(ctx Context) {
		ctx.Next()
		if !ctx.ResponseWriter().HeaderWritten() {
			err := ctx.State().Error
			httpStatus := ctx.State().HttpStatus
			if httpStatus == 0 {
				if err != nil {
					if err.Type() == errx.TypeInternal {
						httpStatus = http.StatusInternalServerError
					} else {
						httpStatus = http.StatusBadRequest
					}
				} else {
					httpStatus = http.StatusOK
				}
			}
			if ctx.ResponseContentType() != "" {
				ctx.ResponseWriter().Header().Set("Content-Type", string(ctx.ResponseContentType()))
			}
			ctx.ResponseWriter().WriteHeader(httpStatus)
		}
		if !ctx.ResponseWriter().BodyWritten() {
			data := ctx.State().Data
			if ctx.ResponseContentType() == api.ContentTypeApplicationJson && ctx.RequireWrapOutput() {
				err := ctx.State().Error
				w := JsonResponseWrapper{
					Success: err == nil,
					Data:    data,
				}
				if err != nil {
					ed := &ErrorDetails{
						Code: err.Code(),
						Type: err.Type(),
					}
					if err.Type() != errx.TypeInternal {
						ed.Message = err.Error()
					}
					w.Error = ed
				}
				data, err = util.Json().Marshal(w)
				if err != nil {
					logrus.WithContext(ctx).WithError(err).Error("failed to marshal json response")
				}
			}
			_, err := ctx.ResponseWriter().Write(data)
			if err != nil {
				logrus.WithContext(ctx).WithError(err).Error("write response failed")
			}
		}
	}
}

func NotFoundHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		fields := logrus.Fields{
			"endpoint": fmt.Sprintf("%s %s", req.Method, req.URL.Path),
		}
		if q := req.URL.Query(); len(q) > 0 {
			fields["query"] = q
		}
		if logrus.GetLevel() > logrus.InfoLevel {
			fields["requestHeader"] = req.Header
			if body, err := ReadBodyReusable(req); err == nil {
				fields["requestBody"] = string(body)
			}
		}
		writer.WriteHeader(http.StatusNotFound)
		logrus.WithFields(fields).WithContext(req.Context()).Warn("route not found")
	}
}
