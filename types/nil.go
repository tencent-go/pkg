package types

import "reflect"

type Nil struct{}

func (Nil) IsNil() {}

type iNil interface {
	IsNil()
}

var nilInterface = reflect.TypeOf((*iNil)(nil)).Elem()

func IsNilType(t reflect.Type) bool {
	return t.Implements(nilInterface)
}

func IsNilValue[T any](t T) bool {
	_, ok := any(t).(iNil)
	return ok
}
