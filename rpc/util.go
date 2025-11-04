package rpc

import (
  "bytes"
  "io"
  "net/http"
  "strconv"

  "github.com/tencent-go/pkg/ctxx"
  "github.com/tencent-go/pkg/errx"
  "github.com/tencent-go/pkg/types"
  "github.com/sirupsen/logrus"
  "github.com/vmihailenco/msgpack/v5"
)

func WriteError(w http.ResponseWriter, err errx.Error) {
  switch err.Type() {
  case errx.TypeAuthorization:
    w.WriteHeader(http.StatusForbidden)
  case errx.TypeAuthentication:
    w.WriteHeader(http.StatusUnauthorized)
  case errx.TypeNotFound:
    w.WriteHeader(http.StatusNotFound)
  default:
    w.WriteHeader(http.StatusBadGateway)
  }
  detail := ErrorDetail{
    Message: err.Error(),
    Type:    err.Type(),
    Code:    err.Code(),
  }
  encoder := msgpack.GetEncoder()
  defer msgpack.PutEncoder(encoder)
  encoder.Reset(w)
  encoder.SetCustomStructTag("json")
  if e := encoder.Encode(detail); e != nil {
    logrus.WithError(e).Error()
    return
  }
}

func WriteSuccess[T any](w http.ResponseWriter, data *T) errx.Error {
  w.WriteHeader(http.StatusOK)
  if data != nil {
    encoder := msgpack.GetEncoder()
    defer msgpack.PutEncoder(encoder)
    encoder.Reset(w)
    encoder.SetCustomStructTag("json")
    if e := encoder.Encode(data); e != nil {
      return errx.Wrap(e).Err()
    }
  }
  return nil
}

func ReadRequestBody[T any](r *http.Request) (*T, errx.Error) {
  var res T
  defer func() { _ = r.Body.Close() }()
  if types.IsNilValue(res) {
    return &res, nil
  }
  if r.ContentLength == 0 {
    return nil, errx.New("request content length is zero")
  }
  decoder := msgpack.GetDecoder()
  defer msgpack.PutDecoder(decoder)
  decoder.Reset(r.Body)
  decoder.SetCustomStructTag("json")
  if e := decoder.Decode(&res); e != nil {
    return nil, errx.Wrap(e).Err()
  }
  return &res, nil
}

func NewRequestBody[T any](value T) (io.Reader, errx.Error) {
  var body io.Reader
  if !types.IsNilValue(value) {
    var buf bytes.Buffer
    encoder := msgpack.GetEncoder()
    encoder.Reset(&buf)
    encoder.SetCustomStructTag("json")
    defer msgpack.PutEncoder(encoder)
    if e := encoder.Encode(value); e != nil {
      return nil, errx.Wrap(e).Err()
    }
    body = &buf
  }
  return body, nil
}

func WriteRequestHeader(ctx ctxx.Context, r *http.Request) {
  h := r.Header
  h.Set("rpc-trace-id", strconv.FormatInt(int64(ctx.GetTraceID()), 10))
  h.Set("rpc-caller", ctx.GetCaller())
  h.Set("rpc-operator", ctx.GetOperator())
  h.Set("rpc-locale", string(ctx.GetLocale()))
  h.Set("Content-Type", "application/msgpack")
}

func ParseResponse[T any](resp *http.Response) (*T, errx.Error) {
  defer func() { _ = resp.Body.Close() }()
  if resp.StatusCode >= 400 {
    detail := ErrorDetail{}
    decoder := msgpack.GetDecoder()
    defer msgpack.PutDecoder(decoder)
    decoder.Reset(resp.Body)
    decoder.SetCustomStructTag("json")
    if e := decoder.Decode(&detail); e != nil {
      return nil, errx.Wrap(e).Err()
    }
    return nil, errx.Define().WithMsg(detail.Message).WithType(detail.Type).WithCode(detail.Code).Err()
  }
  var output T
  if types.IsNilValue(output) {
    return &output, nil
  }
  if resp.ContentLength == 0 {
    return nil, errx.New("response content length is zero")
  }
  decoder := msgpack.GetDecoder()
  defer msgpack.PutDecoder(decoder)
  decoder.Reset(resp.Body)
  decoder.SetCustomStructTag("json")
  if e := decoder.Decode(&output); e != nil {
    return nil, errx.Wrap(e).Err()
  }
  return &output, nil
}

type ErrorDetail struct {
  Message string    `json:"message"`
  Type    errx.Type `json:"type"`
  Code    int       `json:"code"`
}
