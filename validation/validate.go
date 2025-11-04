package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/tencent-go/pkg/errx"
)

func ValidateStructWithCache[T any](value T, opts ...Option) errx.Error {
	v := reflect.ValueOf(value)
	c, ok, err := getOrCreateConfig(v.Type(), opts...)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return c.Validate(v)
}

func (c *Config) Validate(value reflect.Value, parentLabels ...string) (err errx.Error) {
	labels := parentLabels
	if c.label != "" {
		labels = append(labels, c.label)
	}
	defer func() {
		if err == nil {
			return
		}
		if len(labels) > 0 {
			err = errx.Wrap(err).AppendMsgf("Field '%s'", strings.Join(labels, ".")).Err()
		}
	}()
	if !value.IsValid() {
		return errx.New("value is not valid")
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			if c.required {
				return errx.Validation.WithMsg("value is required").Err()
			}
			return nil
		}
		value, err = dereference(value)
		if err != nil {
			return err
		}
		return c.runValidators(value, labels...)
	}
	if c.required {
		if c.hasValidators() {
			return c.runValidators(value, labels...)
		}
		if isZero(value) {
			return errx.Validation.WithMsg("value is required").Err()
		}
		return nil
	}
	if isZero(value) {
		return nil
	}
	return c.runValidators(value, labels...)
}

func (c *Config) runValidators(value reflect.Value, labels ...string) errx.Error {
	for _, validator := range c.validators {
		if err := validator(value); err != nil {
			return err
		}
	}
	if c.mapConfig != nil && value.Kind() == reflect.Map {
		for _, key := range value.MapKeys() {
			l := labels
			{
				k, e := dereference(key)
				if e != nil {
					return e
				}
				l = append(labels, fmt.Sprint(k.Interface()))
			}
			if conf := c.mapConfig.KeyConfig; conf != nil {
				if err := conf.Validate(key, l...); err != nil {
					return err
				}
			}
			if conf := c.mapConfig.ValueConfig; conf != nil {
				if err := conf.Validate(value.MapIndex(key), l...); err != nil {
					return err
				}
			}
		}
	}
	if c.arrayConfig != nil && (value.Kind() == reflect.Slice || value.Kind() == reflect.Array) {
		for i := 0; i < value.Len(); i++ {
			l := append(labels, fmt.Sprintf("%d", i))
			if err := c.arrayConfig.Validate(value.Index(i), l...); err != nil {
				return err
			}
		}
	}
	if c.structFieldsConfig != nil && value.Kind() == reflect.Struct {
		for _, conf := range c.structFieldsConfig {
			if conf.Index < 0 || conf.Index >= value.NumField() {
				return errx.Newf("invalid struct field index: %d", conf.Index)
			}
			if conf.Config == nil {
				return errx.Newf("invalid struct field config: %d", conf.Index)
			}
			if err := conf.Config.Validate(value.Field(conf.Index), labels...); err != nil {
				return err
			}
		}
	}
	return nil
}

func dereference(value reflect.Value) (reflect.Value, errx.Error) {
	if !value.IsValid() {
		return reflect.Value{}, errx.New("Value is not valid")
	}
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return reflect.Value{}, errx.New("Value is nil")
		}
		value = value.Elem() // 解引用指針
	}
	return value, nil
}

type ZeroChecker interface {
	IsZero() bool
}

var zeroCheckerInterface = reflect.TypeOf((*ZeroChecker)(nil)).Elem()

func isZero(value reflect.Value) bool {
	if value.Type().Implements(zeroCheckerInterface) {
		return value.Interface().(ZeroChecker).IsZero()
	}
	if value.Kind() == reflect.Bool {
		return false
	}
	return value.IsZero()
}
