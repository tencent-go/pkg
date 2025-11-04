package errx

import (
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

type Error interface {
	error
	fmt.Formatter
	Unwrap() error
	Cause() error
	Code() int
	Type() Type
	Stack() Stack
}

type Builder interface {
	WithMsg(string) Builder
	WithMsgf(format string, a ...any) Builder
	AppendMsg(string) Builder
	AppendMsgf(format string, a ...any) Builder
	WithCode(int) Builder
	WithType(Type) Builder
	Err() Error
}

type Type string

const (
	TypeInternal       Type = "internal"
	TypeNotFound       Type = "not_found"
	TypeValidation     Type = "validation"
	TypeAuthentication Type = "authentication"
	TypeAuthorization  Type = "authorization"
	TypeRateLimit      Type = "rate_limit"
	TypeNetwork        Type = "network"
	TypeTimeout        Type = "timeout"
	TypeConcurrency    Type = "concurrency"
	TypeBusiness       Type = "business"
	TypeConflict       Type = "conflict"
)

type Frame struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
}

type Stack []Frame

func (f Frame) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		switch {
		case s.Flag('+'):
			_, _ = io.WriteString(s, f.Name)
			_, _ = io.WriteString(s, "\n\t")
			_, _ = io.WriteString(s, f.File)
		default:
			_, _ = io.WriteString(s, path.Base(f.File))
		}
	case 'd':
		_, _ = io.WriteString(s, strconv.Itoa(f.Line))
	case 'n':
		_, _ = io.WriteString(s, funcname(f.Name))
	case 'v':
		f.Format(s, 's')
		_, _ = io.WriteString(s, ":")
		f.Format(s, 'd')
	}
}

func funcname(name string) string {
	i := strings.LastIndex(name, "/")
	name = name[i+1:]
	i = strings.Index(name, ".")
	return name[i+1:]
}

func (s Stack) Format(st fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case st.Flag('+'):
			for _, f := range s {
				_, _ = fmt.Fprintf(st, "\n%+v", f)
			}
		}
	}
}
