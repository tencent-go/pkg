package validation

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/tencent-go/pkg/util"

	"github.com/tencent-go/pkg/errx"
)

type Config struct {
	required           bool
	label              string
	mapConfig          *MapConfig
	arrayConfig        *Config
	structFieldsConfig []StructFieldConfig
	validators         []Validator
}

func (c *Config) IsRequired() bool {
	return c.required
}

func (c *Config) GetLabel() string {
	return c.label
}

type MapConfig struct {
	KeyConfig   *Config
	ValueConfig *Config
}

type StructFieldConfig struct {
	Index  int
	Config *Config
}

type options struct {
	alwaysValidate bool
	labelTags      map[string]bool
}

type Option func(*options)

func AlwaysValidate() Option {
	return func(o *options) {
		o.alwaysValidate = true
	}
}

func WithLabelTag(labelTag string) Option {
	return func(o *options) {
		o.labelTags[labelTag] = true
	}
}

func WithLabelTags(labelTags ...string) Option {
	return func(o *options) {
		for _, tag := range labelTags {
			o.labelTags[tag] = true
		}
	}
}

func GetOrCreateValidator(typ reflect.Type, opts ...Option) (func(value any) errx.Error, bool, errx.Error) {
	if typ.Implements(validatableInterface) {
		return func(value any) errx.Error {
			return value.(Validatable).Validate()
		}, true, nil
	}
	config, ok, err := getOrCreateConfig(typ, opts...)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	return func(value any) errx.Error {
		return config.Validate(reflect.ValueOf(value))
	}, true, nil
}

var sortedTags = map[string]int{
	"query":  0,
	"path":   1,
	"header": 2,
	"env":    3,
	"json":   4,
	"form":   5,
}

func getOrCreateConfig(typ reflect.Type, opts ...Option) (*Config, bool, errx.Error) {
	o := &options{
		labelTags: map[string]bool{},
	}
	for _, opt := range opts {
		opt(o)
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	var key string
	if len(o.labelTags) > 0 {
		tags := make([]string, 0, len(o.labelTags))
		for t := range o.labelTags {
			tags = append(tags, t)
		}
		sort.Strings(tags)
		key = fmt.Sprintf("%s.%s.%s.%t", typ.Kind(), typ.Name(), strings.Join(tags, "_"), o.alwaysValidate)
	} else {
		key = fmt.Sprintf("%s.%s.%t", typ.Kind(), typ.Name(), o.alwaysValidate)
	}
	res, exists := cachedConfig.Load(key)
	if exists {
		if res == nil {
			return nil, false, nil
		} else {
			return res, true, nil
		}
	}
	c, ok, err := CreateConfig(typ, Rule{}, o.alwaysValidate, o.labelTags)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		c = nil
	}
	res, _ = cachedConfig.LoadOrStore(key, c)
	return res, res != nil, nil
}

var cachedConfig = util.LazyMap[string, *Config]{}

func (c *Config) hasValidators() bool {
	return len(c.validators) > 0 || c.mapConfig != nil || c.arrayConfig != nil || c.structFieldsConfig != nil
}

var validatableInterface = reflect.TypeOf((*Validatable)(nil)).Elem()

func CreateConfig(typ reflect.Type, rule Rule, alwaysValidate bool, labelTags map[string]bool) (res *Config, ok bool, err errx.Error) {
	res = &Config{}
	defer func() {
		if err != nil {
			res = nil
			ok = false
			return
		}
		if !res.required && !res.hasValidators() {
			res = nil
			ok = false
			return
		}
		ok = true
	}()
	if rule.Label != nil {
		res.label = *rule.Label
	}
	if rule.Required != nil {
		res.required = *rule.Required
	} else {
		res.required = true
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Implements(validatableInterface) {
		res.validators = append(res.validators, validatableValidator)
		return
	}

	for _, builder := range customValidatorBuilders {
		if validator, ok := builder(typ, &rule); ok {
			res.validators = append(res.validators, validator)
		}
	}

	if len(res.validators) > 0 {
		return
	}

	for _, builder := range defaultValidatorBuilders {
		if validator, ok := builder(typ, &rule); ok {
			res.validators = append(res.validators, validator)
		}
	}

	if typ.Kind() == reflect.Struct {
		res.structFieldsConfig, err = CreateStructConfig(typ, alwaysValidate, labelTags)
		return
	}

	if typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
		o := rule.Dive
		if o == nil {
			o = &Rule{}
		}
		res.arrayConfig, _, err = CreateConfig(typ.Elem(), *o, alwaysValidate, labelTags)
		return
	}

	if typ.Kind() == reflect.Map {
		var keyConfig *Config
		var valueConfig *Config
		{
			o := rule.MapKey
			if o == nil {
				o = &Rule{}
			}
			if o != nil {
				keyConfig, _, err = CreateConfig(typ.Key(), *o, alwaysValidate, labelTags)
				if err != nil {
					return
				}
			}
		}
		{
			o := rule.Dive
			if o == nil {
				o = &Rule{}
			}
			if o != nil {
				valueConfig, _, err = CreateConfig(typ.Elem(), *o, alwaysValidate, labelTags)
				if err != nil {
					return
				}
			}
		}
		if keyConfig != nil || valueConfig != nil {
			res.mapConfig = &MapConfig{
				KeyConfig:   keyConfig,
				ValueConfig: valueConfig,
			}
		}
		return
	}
	return
}

func CreateStructConfig(typ reflect.Type, alwaysValidate bool, labelTags map[string]bool) ([]StructFieldConfig, errx.Error) {
	if typ.Kind() != reflect.Struct {
		return nil, errx.Newf("%s is not a struct", typ.Name())
	}
	var res []StructFieldConfig
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if len(labelTags) == 0 {
			fieldConfig, ok, err := CreateStructFieldConfig(field, alwaysValidate, "")
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			c := &StructFieldConfig{
				Index:  i,
				Config: fieldConfig,
			}
			res = append(res, *c)
		} else {
			tags := make([]string, 0, len(labelTags))
			for tag := range labelTags {
				tags = append(tags, tag)
			}
			sort.Slice(tags, func(i, j int) bool {
				oi, ok := sortedTags[tags[i]]
				if !ok {
					oi = 99999
				}
				oj, ok := sortedTags[tags[j]]
				if !ok {
					oj = 99999
				}
				return oi < oj
			})
			for _, tag := range tags {
				fieldConfig, ok, err := CreateStructFieldConfig(field, alwaysValidate, tag)
				if err != nil {
					return nil, err
				}
				if !ok {
					continue
				}
				c := &StructFieldConfig{
					Index:  i,
					Config: fieldConfig,
				}
				res = append(res, *c)
				break //僅作用與第一個
			}
		}
	}
	return res, nil
}

func CreateStructFieldConfig(field reflect.StructField, alwaysValidate bool, labelTag string) (*Config, bool, errx.Error) {
	if !field.IsExported() {
		return nil, false, nil
	}
	var vTag, lTag string
	vTag = field.Tag.Get("validate")
	if labelTag != "" {
		lTag = field.Tag.Get(labelTag)
	}

	{
		var ignore bool
		if len(vTag) > 0 && (vTag[0] == '-' || strings.HasPrefix(vTag, "ignore")) {
			ignore = true
		}
		if len(lTag) > 0 && lTag[0] == '-' {
			ignore = true
		}
		if ignore {
			return nil, false, nil
		}
	}
	labelTags := map[string]bool{}
	if labelTag != "" {
		labelTags[field.Name] = true
	}
	if field.Anonymous {
		fields, err := CreateStructConfig(field.Type, alwaysValidate, labelTags)
		if err != nil {
			return nil, false, err
		}
		if len(fields) == 0 {
			return nil, false, nil
		}
		return &Config{
			structFieldsConfig: fields,
		}, true, nil
	}

	if !alwaysValidate && vTag == "" && lTag == "" {
		return nil, false, nil
	}

	rule := Rule{}
	if lTag != "" {
		if strings.HasSuffix(lTag, ",omitempty") {
			lTag = lTag[:len(lTag)-len(",omitempty")]
			rule.SetRequired(false)
		}
		if len(lTag) > 0 {
			rule.SetLabel(lTag)
		}
	}
	if vTag != "" {
		if err := rule.Parse(vTag); err != nil {
			return nil, false, err
		}
	}
	if rule.Label == nil {
		rule.SetLabel(field.Name)
	}
	return CreateConfig(field.Type, rule, alwaysValidate, labelTags)
}
