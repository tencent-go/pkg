package errx

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type impl struct {
	cause error
	msg   string
	code  int
	stack Stack
	typ   Type
}

func (i *impl) Error() string {
	return i.msg
}

func (i *impl) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			i.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, i.msg)
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", i.msg)
	}
}

func (i *impl) Unwrap() error {
	return i.cause
}

func (i *impl) Cause() error {
	return errors.Unwrap(i)
}

func (i *impl) Code() int {
	return i.code
}

func (i *impl) Type() Type {
	return i.typ
}

func (i *impl) Stack() Stack {
	return i.stack
}

func (i *impl) copy() *impl {
	return &impl{
		cause: i,
		msg:   i.msg,
		code:  i.code,
		stack: i.stack,
		typ:   i.typ,
	}
}

func (i *impl) WithMsg(s string) Builder {
	c := i.copy()
	c.msg = s
	return c
}

func (i *impl) WithMsgf(format string, a ...any) Builder {
	c := i.copy()
	c.msg = fmt.Sprintf(format, a...)
	return c
}

func (i *impl) AppendMsg(appendMsg string) Builder {
	c := i.copy()
	if c.msg == "" {
		c.msg = appendMsg
	} else {
		c.msg = fmt.Sprintf("%s: %s", appendMsg, c.msg)
	}
	return c
}

func (i *impl) AppendMsgf(format string, a ...any) Builder {
	c := i.copy()
	appendMsg := fmt.Sprintf(format, a...)
	if c.msg == "" {
		c.msg = appendMsg
	} else {
		c.msg = fmt.Sprintf("%s: %s", appendMsg, c.msg)
	}
	return c
}

func (i *impl) WithCode(code int) Builder {
	c := i.copy()
	c.code = code
	return c
}

func (i *impl) WithType(t Type) Builder {
	c := i.copy()
	c.typ = t
	return c
}

func (i *impl) Err() Error {
	c := i.copy()
	if c.stack == nil {
		c.stack = callers()
	}
	return c
}

var framePattern = regexp.MustCompile(`(?m)(?P<Name>.+)\n\t(?P<File>.+):(?P<Line>\d+)`)

func parseStack(stackStr string) Stack {
	var stack Stack
	matches := framePattern.FindAllStringSubmatch(stackStr, -1)
	for _, match := range matches {
		line, _ := strconv.Atoi(match[3])
		frame := Frame{
			Name: match[1],
			File: match[2],
			Line: line,
		}
		stack = append(stack, frame)
	}
	if len(stack) == 0 {
		return nil
	}
	return stack
}

func callers() Stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:]) // skip first 3 callers
	frames := runtime.CallersFrames(pcs[:n])
	var stack Stack
	for {
		frame, more := frames.Next()
		if strings.Contains(frame.Function, "go-pkg/errx") {
			continue
		}
		stack = append(stack, Frame{
			Name: frame.Function,
			File: frame.File,
			Line: frame.Line,
		})
		if !more {
			break
		}
	}
	return stack
}
