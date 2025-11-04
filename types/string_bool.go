package types

import (
	"github.com/tencent-go/pkg/errx"
)

type StringBool string

const (
	StringBoolTrue  StringBool = "true"
	StringBoolFalse StringBool = "false"
)

func (s StringBool) Bool() bool {
	return s == StringBoolTrue
}

func (s StringBool) String() string {
	return string(s)
}

func (s StringBool) Validate() errx.Error {
	if s != StringBoolTrue && s != StringBoolFalse {
		return errx.Validation.WithMsgf("invalid string bool value: %s", s).Err()
	}
	return nil
}

func (s StringBool) Values() []string {
	return []string{StringBoolTrue.String(), StringBoolFalse.String()}
}
