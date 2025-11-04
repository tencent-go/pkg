package errx

import (
	"context"
	"errors"
	"fmt"
)

var (
	Validation     = Define().WithType(TypeValidation).WithMsg("validation failed")
	Authentication = Define().WithType(TypeAuthentication).WithMsg("unauthenticated")
	Authorization  = Define().WithType(TypeAuthorization).WithMsg("unauthorized")
	NotFound       = Define().WithType(TypeNotFound).WithMsg("not found")
	Concurrency    = Define().WithType(TypeConcurrency).WithMsg("concurrency")
	Conflict       = Define().WithType(TypeConflict).WithMsg("conflict")
	Business       = Define().WithType(TypeBusiness).WithMsg("business error")
	Internal       = Define().WithMsg("internal error")
)

func Define() Builder {
	return rootError
}

var rootError = &impl{
	cause: errors.New(""),
	typ:   TypeInternal,
}

type emptyError struct{}

func (e *emptyError) WithMsg(s string) Builder {
	return rootError.WithMsg(s)
}

func (e *emptyError) WithMsgf(format string, a ...any) Builder {
	return rootError.WithMsgf(format, a...)
}

func (e *emptyError) AppendMsg(s string) Builder {
	return rootError.AppendMsg(s)
}

func (e *emptyError) AppendMsgf(format string, a ...any) Builder {
	return rootError.AppendMsgf(format, a...)
}

func (e *emptyError) WithCode(i int) Builder {
	return rootError.WithCode(i)
}

func (e *emptyError) WithType(t Type) Builder {
	return rootError.WithType(t)
}

func (e *emptyError) Err() Error {
	return nil
}

var empty = &emptyError{}

func Wrap(err error) Builder {
	if err == nil {
		return empty
	}
	if i, ok := err.(*impl); ok {
		return i
	}
	if e, ok := err.(Error); ok {
		return &impl{
			cause: e,
			msg:   e.Error(),
			stack: e.Stack(),
			code:  e.Code(),
			typ:   e.Type(),
		}
	}
	t := TypeInternal
	if errors.Is(err, context.DeadlineExceeded) {
		t = TypeTimeout
	} else if errors.Is(err, context.Canceled) {
		t = TypeBusiness
	}
	return &impl{
		cause: err,
		typ:   t,
		msg:   err.Error(),
		stack: parseStack(fmt.Sprintf("%+v", err)),
	}
}

func New(msg string) Error {
	return Define().WithMsg(msg).Err()
}

func Newf(format string, a ...any) Error {
	return Define().WithMsgf(format, a...).Err()
}

func C(code int, msg string) Error {
	return Define().WithCode(code).WithMsg(msg).Err()
}

func Cf(code int, format string, a ...any) Error {
	return Define().WithCode(code).WithMsgf(format, a...).Err()
}
