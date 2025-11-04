package validation

import (
	"reflect"
	"strconv"
	"time"

	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/types"
	"github.com/shopspring/decimal"
)

type Validatable interface {
	Validate() errx.Error
}

type Validator func(value reflect.Value) errx.Error

type ValidatorBuilder func(typ reflect.Type, rule *Rule) (Validator, bool)

var defaultValidatorBuilders = []ValidatorBuilder{
	intRangeValidatorBuilder,
	decimalRangeValidatorBuilder,
	floatRangeValidatorBuilder,
	timeRangeValidatorBuilder,
	lengthRangeValidatorBuilder,
	patternValidatorBuilder,
	enumValidatorBuilder,
}

var customValidatorBuilders []ValidatorBuilder

func RegisterValidatorBuilder(builder ValidatorBuilder) {
	customValidatorBuilders = append(customValidatorBuilders, builder)
}

func validatableValidator(value reflect.Value) errx.Error {
	if v, ok := value.Interface().(Validatable); ok {
		return v.Validate()
	}
	return errx.Newf("value is not validatable")
}

func intRangeValidatorBuilder(typ reflect.Type, rule *Rule) (Validator, bool) {
	if typ.Kind() < reflect.Int || typ.Kind() > reflect.Uint64 {
		return nil, false
	}
	var minValue, maxValue *int64
	if rule.Min != nil {
		cond, err := strconv.ParseInt(*rule.Min, 10, 64)
		if err != nil {
			return nil, false
		}
		minValue = &cond
	}
	if rule.Max != nil {
		cond, err := strconv.ParseInt(*rule.Max, 10, 64)
		if err != nil {
			return nil, false
		}
		maxValue = &cond
	}
	if minValue != nil && maxValue != nil {
		return func(value reflect.Value) errx.Error {
			if value.Int() < *minValue || value.Int() > *maxValue {
				return errx.Validation.WithMsgf("value must be greater than or equal to %d and less than or equal to %d.", *minValue, *maxValue).Err()
			}
			return nil
		}, true
	}
	if minValue != nil {
		return func(value reflect.Value) errx.Error {
			if value.Int() < *minValue {
				return errx.Validation.WithMsgf("value must be greater than or equal to %d.", *minValue).Err()
			}
			return nil
		}, true
	}
	if maxValue != nil {
		return func(value reflect.Value) errx.Error {
			if value.Int() > *maxValue {
				return errx.Validation.WithMsgf("value must be less than or equal to %d.", *maxValue).Err()
			}
			return nil
		}, true
	}
	return nil, false
}

func decimalRangeValidatorBuilder(typ reflect.Type, rule *Rule) (Validator, bool) {
	if !typ.ConvertibleTo(decimalType) {
		return nil, false
	}
	var minValue, maxValue *decimal.Decimal
	if rule.Min != nil {
		cond, err := decimal.NewFromString(*rule.Min)
		if err == nil {
			minValue = &cond
		}
	}
	if rule.Max != nil {
		cond, err := decimal.NewFromString(*rule.Max)
		if err == nil {
			maxValue = &cond
		}
	}
	if minValue != nil && maxValue != nil {
		return func(value reflect.Value) errx.Error {
			d, ok := value.Interface().(decimal.Decimal)
			if !ok {
				return errx.Newf("value is not decimal")
			}
			if d.LessThan(*minValue) || d.GreaterThan(*maxValue) {
				return errx.Validation.WithMsgf("value must be greater than or equal to %s and less than or equal to %s.", *minValue, *maxValue).Err()
			}
			return nil
		}, true
	}
	if minValue != nil {
		return func(value reflect.Value) errx.Error {
			d, ok := value.Interface().(decimal.Decimal)
			if !ok {
				return errx.Newf("value is not decimal")
			}
			if d.LessThan(*minValue) {
				return errx.Validation.WithMsgf("value must be greater than or equal to %s.", *minValue).Err()
			}
			return nil
		}, true
	}
	if maxValue != nil {
		return func(value reflect.Value) errx.Error {
			d, ok := value.Interface().(decimal.Decimal)
			if !ok {
				return errx.Newf("value is not decimal")
			}
			if d.GreaterThan(*maxValue) {
				return errx.Validation.WithMsgf("value must be less than or equal to %s.", *maxValue).Err()
			}
			return nil
		}, true
	}
	return nil, false
}

func floatRangeValidatorBuilder(typ reflect.Type, rule *Rule) (Validator, bool) {
	if typ.Kind() != reflect.Float32 && typ.Kind() != reflect.Float64 {
		return nil, false
	}
	var minValue, maxValue *float64
	if rule.Min != nil {
		cond, err := strconv.ParseFloat(*rule.Min, 64)
		if err != nil {
			return nil, false
		}
		minValue = &cond
	}
	if rule.Max != nil {
		cond, err := strconv.ParseFloat(*rule.Max, 64)
		if err != nil {
			return nil, false
		}
		maxValue = &cond
	}
	if minValue != nil && maxValue != nil {
		return func(value reflect.Value) errx.Error {
			f := value.Float()
			if f < *minValue || f > *maxValue {
				return errx.Validation.WithMsgf("value must be greater than or equal to %f and less than or equal to %f.", *minValue, *maxValue).Err()
			}
			return nil
		}, true
	}
	if minValue != nil {
		return func(value reflect.Value) errx.Error {
			f := value.Float()
			if f < *minValue {
				return errx.Validation.WithMsgf("value must be greater than or equal to %f.", *minValue).Err()
			}
			return nil
		}, true
	}
	if maxValue != nil {
		return func(value reflect.Value) errx.Error {
			f := value.Float()
			if f > *maxValue {
				return errx.Validation.WithMsgf("value must be less than or equal to %f.", *maxValue).Err()
			}
			return nil
		}, true
	}
	return nil, false
}

func timeRangeValidatorBuilder(typ reflect.Type, rule *Rule) (Validator, bool) {
	if !typ.ConvertibleTo(timeType) {
		return nil, false
	}
	var minTime, maxTime *time.Time
	if rule.Min != nil {
		cond, err := time.Parse(time.RFC3339, *rule.Min)
		if err == nil {
			minTime = &cond
		}
	}
	if rule.Max != nil {
		cond, err := time.Parse(time.RFC3339, *rule.Max)
		if err == nil {
			maxTime = &cond
		}
	}
	if minTime != nil && maxTime != nil {
		return func(value reflect.Value) errx.Error {
			t, ok := value.Interface().(time.Time)
			if !ok {
				return errx.Newf("value is not time")
			}
			if t.Before(*minTime) || t.After(*maxTime) {
				return errx.Validation.WithMsgf("value must be greater than or equal to %s and less than or equal to %s.", *minTime, *maxTime).Err()
			}
			return nil
		}, true
	}
	if minTime != nil {
		return func(value reflect.Value) errx.Error {
			t, ok := value.Interface().(time.Time)
			if !ok {
				return errx.Newf("value is not time")
			}
			if t.Before(*minTime) {
				return errx.Validation.WithMsgf("value must be greater than or equal to %s.", *minTime).Err()
			}
			return nil
		}, true
	}
	if maxTime != nil {
		return func(value reflect.Value) errx.Error {
			t, ok := value.Interface().(time.Time)
			if !ok {
				return errx.Newf("value is not time")
			}
			if t.After(*maxTime) {
				return errx.Validation.WithMsgf("value must be less than or equal to %s.", *maxTime).Err()
			}
			return nil
		}, true
	}
	return nil, false
}

func lengthRangeValidatorBuilder(typ reflect.Type, rule *Rule) (Validator, bool) {
	if typ.Kind() != reflect.String && typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array && typ.Kind() != reflect.Map {
		return nil, false
	}
	if rule.Min == nil && rule.Max == nil {
		return nil, false
	}
	var minLen, maxLen *int
	if rule.Min != nil {
		cond, err := strconv.Atoi(*rule.Min)
		if err != nil {
			return nil, false
		}
		minLen = &cond
	}
	if rule.Max != nil {
		cond, err := strconv.Atoi(*rule.Max)
		if err != nil {
			return nil, false
		}
		maxLen = &cond
	}
	if minLen != nil && maxLen != nil {
		return func(value reflect.Value) errx.Error {
			l := value.Len()
			if l < *minLen || l > *maxLen {
				return errx.Validation.WithMsgf("value length must be greater than or equal to %d and less than or equal to %d.", *minLen, *maxLen).Err()
			}
			return nil
		}, true
	}
	if minLen != nil {
		return func(value reflect.Value) errx.Error {
			l := value.Len()
			if l < *minLen {
				return errx.Validation.WithMsgf("value length must be greater than or equal to %d.", *minLen).Err()
			}
			return nil
		}, true
	}
	if maxLen != nil {
		return func(value reflect.Value) errx.Error {
			l := value.Len()
			if l > *maxLen {
				return errx.Validation.WithMsgf("value length must be less than or equal to %d.", *maxLen).Err()
			}
			return nil
		}, true
	}
	return nil, false
}

func patternValidatorBuilder(typ reflect.Type, rule *Rule) (Validator, bool) {
	if typ.Kind() != reflect.String {
		return nil, false
	}
	if rule.Pattern == nil {
		return nil, false
	}
	return func(value reflect.Value) errx.Error {
		if !rule.Pattern.MatchString(value.String()) {
			return errx.Validation.WithMsg("format error").Err()
		}
		return nil
	}, true
}

func enumValidatorBuilder(typ reflect.Type, rule *Rule) (Validator, bool) {
	if !typ.Implements(enumInterface) {
		return nil, false
	}
	return func(value reflect.Value) errx.Error {
		if v, ok := value.Interface().(types.IEnum); ok {
			if !v.Enum().Contains(v) {
				return errx.Validation.WithMsgf("value is not valid").Err()
			}
		}
		return nil
	}, true
}

var (
	timeType      = reflect.TypeOf(time.Time{})
	decimalType   = reflect.TypeOf(decimal.Decimal{})
	enumInterface = reflect.TypeOf((*types.IEnum)(nil)).Elem()
)
