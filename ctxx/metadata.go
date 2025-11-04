package ctxx

import (
	"github.com/tencent-go/pkg/env"
	"github.com/tencent-go/pkg/types"
)

type Metadata struct {
	TraceID  types.ID     `json:"traceId"`
	Operator string       `json:"operator"`
	Caller   string       `json:"caller"`
	Locale   types.Locale `json:"locale"`
}

func (m *Metadata) FillDefaults() {
	if m.TraceID == 0 {
		m.TraceID = types.NewID()
	}
	if m.Caller == "" {
		c := env.BaseConfigReaderBuilder.Build().Read()
		caller := c.PodName
		if caller == "" {
			caller = "unknown"
		}
		m.Caller = caller
	}
	if m.Operator == "" {
		m.Operator = "system"
	}
}

func (m *Metadata) GetTraceID() types.ID {
	return m.TraceID
}

func (m *Metadata) GetCaller() string {
	return m.Caller
}

func (m *Metadata) GetLocale() types.Locale {
	return m.Locale
}

func (m *Metadata) GetOperator() string {
	return m.Operator
}
