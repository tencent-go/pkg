package initialization

import (
	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/errx"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"strings"
	"sync"
)

func applyLogSetting(level logrus.Level) {
	logrus.AddHook(&hook{})
	logrus.SetLevel(level)
	logrus.SetFormatter(&jsonFormatter{})
	logrus.SetReportCaller(true)
}

var json = sync.OnceValue(func() jsoniter.API {
	return jsoniter.Config{
		EscapeHTML:                    true,
		SortMapKeys:                   true,
		ValidateJsonRawMessage:        true,
		ObjectFieldMustBeSimpleString: true,
	}.Froze()
})

type jsonFormatter struct {
}

func (j *jsonFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := map[string]interface{}{}
	for s := range entry.Data {
		data[s] = entry.Data[s]
	}
	fs := strings.Split(entry.Caller.File, "/")
	if len(fs) > 3 {
		fs = fs[len(fs)-3:]
	}
	data["file"] = fmt.Sprintf("%s:%d", strings.Join(fs, "/"), entry.Caller.Line)
	data["level"] = strings.ToUpper(entry.Level.String())
	data["message"] = entry.Message
	if entry.Context != nil {
		if ctx, ok := entry.Context.(ctxx.Context); ok {
			data["traceID"] = ctx.GetTraceID().String()
			data["operator"] = ctx.GetOperator()
			data["caller"] = ctx.GetCaller()
		}
	}
	str, err := json().Marshal(data)
	if err != nil {
		return nil, err
	}
	return append(str, '\n'), nil
}

type hook struct{}

func (h *hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *hook) Fire(entry *logrus.Entry) error {
	if entry.Level > logrus.WarnLevel {
		return nil
	}
	if err, withErr := entry.Data[logrus.ErrorKey]; withErr {
		if e, ok := err.(error); ok {
			entry.Data["error"] = e.Error()
		}
		if errx, ok := err.(errx.Error); ok {
			entry.Data["errorType"] = errx.Type()
			entry.Data["errorCode"] = errx.Code()
			stack := errx.Stack()
			if len(stack) > 8 {
				stack = stack[:8]
			}
			entry.Data["stack"] = stack
			return nil
		}
		if errs, ok := err.(xerrors.Formatter); ok {
			entry.Data["stack"] = fmt.Sprintf("%+v", errs)
			return nil
		}
	}
	return nil
}
