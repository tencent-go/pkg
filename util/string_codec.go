package util

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/tencent-go/pkg/errx"
	"github.com/shopspring/decimal"
)

type ReflectStringCodec interface {
	Test(typ reflect.Type) bool
	ToString(value reflect.Value) (string, errx.Error)
	FromString(value reflect.Value, str string) errx.Error
}

type reflectStringCodec struct {
	test       func(typ reflect.Type) bool
	toString   func(value reflect.Value) (string, errx.Error)
	fromString func(value reflect.Value, str string) errx.Error
}

func (r *reflectStringCodec) Test(typ reflect.Type) bool {
	return r.test(typ)
}
func (r *reflectStringCodec) ToString(value reflect.Value) (string, errx.Error) {
	return r.toString(value)
}
func (r *reflectStringCodec) FromString(value reflect.Value, str string) errx.Error {
	return r.fromString(value, str)
}

func CreateStringCodec(test func(typ reflect.Type) bool, toString func(value reflect.Value) (string, errx.Error), fromString func(value reflect.Value, str string) errx.Error) ReflectStringCodec {
	return &reflectStringCodec{
		test:       test,
		toString:   toString,
		fromString: fromString,
	}
}

var stringCodecs = []ReflectStringCodec{
	stringCodec, boolCodec, intCodec, floatCodec, decimalCodec, timeCodec,
}

func RegisterStringCodec(codecs ...ReflectStringCodec) {
	if len(codecs) == 0 {
		return
	}
	stringCodecs = append(codecs, stringCodecs...)
}

func findStringCodec(t reflect.Type) (ReflectStringCodec, bool) {
	for _, codec := range stringCodecs {
		if codec.Test(t) {
			return codec, true
		}
	}
	return nil, false
}

var stringCodec = CreateStringCodec(
	func(t reflect.Type) bool {
		return t.Kind() == reflect.String
	},
	func(v reflect.Value) (string, errx.Error) {
		return v.String(), nil
	},
	func(v reflect.Value, s string) errx.Error {
		if src := reflect.ValueOf(s); src.Type().ConvertibleTo(v.Type()) {
			v.Set(src.Convert(v.Type()))
			return nil
		}
		return errx.New("Cannot convert string to " + v.Type().String())
	},
)

var boolCodec = CreateStringCodec(
	func(t reflect.Type) bool {
		return t.Kind() == reflect.Bool
	},
	func(v reflect.Value) (string, errx.Error) {
		if v.Bool() {
			return "true", nil
		}
		return "false", nil
	},
	func(v reflect.Value, s string) errx.Error {
		bo := false
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true", "1", "yes", "on", "t", "y":
			bo = true
		case "false", "0", "no", "off", "f", "n":
			bo = false
		default:
			return errx.New("Invalid boolean value: " + s)
		}
		if src := reflect.ValueOf(bo); src.Type().ConvertibleTo(v.Type()) {
			v.Set(src.Convert(v.Type()))
		}
		return nil
	},
)

var intCodec = CreateStringCodec(
	func(t reflect.Type) bool {
		return t.Kind() >= reflect.Int && t.Kind() <= reflect.Int64
	},
	func(v reflect.Value) (string, errx.Error) {
		i := v.Int()
		return strconv.FormatInt(i, 10), nil
	},
	func(v reflect.Value, s string) errx.Error {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return errx.New("ParseInt failed")
		}
		if src := reflect.ValueOf(i); src.Type().ConvertibleTo(v.Type()) {
			v.Set(src.Convert(v.Type()))
		}
		return nil
	},
)

var floatCodec = CreateStringCodec(
	func(t reflect.Type) bool {
		return t.Kind() == reflect.Float32 || t.Kind() == reflect.Float64
	},
	func(v reflect.Value) (string, errx.Error) {
		f := v.Float()
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	},
	func(v reflect.Value, s string) errx.Error {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return errx.New("ParseFloat failed")
		}
		if src := reflect.ValueOf(f); src.Type().ConvertibleTo(v.Type()) {
			v.Set(src.Convert(v.Type()))
		}
		return nil
	},
)

var decimalCodec = CreateStringCodec(
	func(t reflect.Type) bool {
		return t == decimalType
	},
	func(v reflect.Value) (string, errx.Error) {
		d, ok := v.Interface().(decimal.Decimal)
		if !ok {
			return "", errx.New("Not a decimal.Decimal")
		}
		return d.String(), nil
	},
	func(v reflect.Value, s string) errx.Error {
		d, err := decimal.NewFromString(s)
		if err != nil {
			return errx.New("NewFromString failed")
		}
		v.Set(reflect.ValueOf(d))
		return nil
	},
)

var timeCodec = CreateStringCodec(
	func(t reflect.Type) bool {
		return t == timeType
	},
	func(v reflect.Value) (string, errx.Error) {
		t, ok := v.Interface().(time.Time)
		if !ok {
			return "", errx.New("Not a time.Time")
		}
		return t.String(), nil
	},
	func(v reflect.Value, s string) errx.Error {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return errx.New("Parse failed")
		}
		v.Set(reflect.ValueOf(t))
		return nil
	},
)

var (
	decimalType = reflect.TypeOf(decimal.Decimal{})
	timeType    = reflect.TypeOf(time.Time{})
)
