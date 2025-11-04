package util

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/tencent-go/pkg/errx"
	"github.com/sirupsen/logrus"
)

type HttpData struct {
	Header http.Header
	Query  url.Values
	Path   map[string]string
	Body   []byte
}

type HttpSerializer[T any] interface {
	Serialize(value T) (*HttpData, errx.Error)
	Deserialize(src *HttpData, dst *T) errx.Error
}

func NewHttpSerializer[T any](tags ...StructTag) (HttpSerializer[T], bool) {
	should := map[StructTag]bool{}
	for _, tag := range tags {
		should[tag] = true
	}
	if len(should) == 0 {
		return nil, false
	}
	c := &structConfig{}
	c.parse(reflect.TypeOf(*(new(T))))
	for tag := range c.taggedFields {
		if !should[tag] {
			delete(c.taggedFields, tag)
		}
	}
	if _, ok := should[TagJson]; ok {
		c.excludeIntersection(TagJson)
	}
	if _, ok := should[TagForm]; ok {
		c.excludeIntersection(TagForm)
	}
	for tag, m := range c.taggedFields {
		if len(m) == 0 {
			delete(c.taggedFields, tag)
		}
	}
	if len(c.taggedFields) == 0 {
		return nil, false
	}
	s := &serializer[T]{c}
	return s, true
}

type serializer[T any] struct {
	*structConfig
}

func (s *serializer[T]) Deserialize(src *HttpData, dst *T) errx.Error {
	if _, ok := s.taggedFields[TagJson]; ok && len(src.Body) > 0 {
		err := Json().Unmarshal(src.Body, dst)
		if err != nil {
			return err
		}
	}
	value := reflect.ValueOf(dst)
	if value.IsNil() {
		return nil
	}
	value = value.Elem()
	if _, ok := s.taggedFields[TagForm]; ok && len(src.Body) > 0 {
		values, e := url.ParseQuery(string(src.Body))
		if e != nil {
			return errx.Wrap(e).Err()
		}
		if err := s.decode(&extValues{values}, TagForm, value); err != nil {
			return err
		}
	}
	if _, ok := s.taggedFields[TagQuery]; ok && len(src.Query) > 0 {
		if err := s.decode(&extValues{src.Query}, TagQuery, value); err != nil {
			return err
		}
	}
	if _, ok := s.taggedFields[TagPath]; ok && len(src.Path) > 0 {
		if err := s.decode(&extPath{src.Path}, TagPath, value); err != nil {
			return err
		}
	}
	if _, ok := s.taggedFields[TagHeader]; ok && len(src.Header) > 0 {
		if err := s.decode(src.Header, TagHeader, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *serializer[T]) Serialize(v T) (*HttpData, errx.Error) {
	data := &HttpData{}
	if _, ok := s.taggedFields[TagJson]; ok {
		// TODO: Fields without json tag and fields with intersection with other tags will also be serialized.
		// But no problematic scenarios have been encountered yet, so no handling for now
		res, err := Json().Marshal(v)
		if err != nil {
			return nil, err
		}
		data.Body = res
	}
	value := reflect.ValueOf(v)
	if _, ok := s.taggedFields[TagForm]; ok {
		val := &extValues{
			make(url.Values),
		}
		if err := s.encode(value, TagForm, val); err != nil {
			return nil, err
		}
		data.Body = []byte(val.Encode())
	}
	if _, ok := s.taggedFields[TagQuery]; ok {
		val := &extValues{
			make(url.Values),
		}
		if err := s.encode(value, TagQuery, val); err != nil {
			return nil, err
		}
		data.Query = val.urlValues
	}
	if _, ok := s.taggedFields[TagPath]; ok {
		val := &extPath{
			make(map[string]string),
		}
		if err := s.encode(value, TagPath, val); err != nil {
			return nil, err
		}
		data.Path = val.p
	}
	if _, ok := s.taggedFields[TagHeader]; ok {
		val := make(http.Header)
		if err := s.encode(value, TagHeader, val); err != nil {
			return nil, err
		}
		data.Header = val
	}
	return data, nil
}

type urlValues = url.Values

type extValues struct {
	urlValues
}

func (ex *extValues) Values(key string) []string {
	return ex.urlValues[key]
}

type extPath struct {
	p map[string]string
}

func (ex *extPath) Values(key string) []string {
	return nil
}

func (ex *extPath) Get(key string) string {
	return ex.p[key]
}

func (ex *extPath) Add(key, value string) {
	ex.p[key] = value
}

type field struct {
	index     []int
	omitempty bool
	codec     ReflectStringCodec
}

type structConfig struct {
	taggedFields map[StructTag]map[string]*field
}

func (s *structConfig) parse(t reflect.Type, parentIdx ...int) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	if s.taggedFields == nil {
		s.taggedFields = make(map[StructTag]map[string]*field)
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		idx := append(parentIdx, i)
		if f.Anonymous {
			s.parse(f.Type, idx...)
			continue
		}
		if !f.IsExported() {
			continue
		}
		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Array || ft.Kind() == reflect.Slice {
			ft = ft.Elem()
			for ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
		}
		codec, _ := findStringCodec(ft)
		for _, tag := range []StructTag{TagHeader, TagQuery, TagPath, TagForm, TagJson} {
			value, ok := f.Tag.Lookup(string(tag))
			if !ok {
				continue
			}
			elements := strings.SplitN(value, ",", 2)
			name := elements[0]
			if name == "-" || name == "" {
				continue
			}
			var omitempty bool
			if len(elements) == 2 {
				omitempty = elements[1] == "omitempty"
			}
			fie := &field{
				index:     idx,
				omitempty: omitempty,
				codec:     codec,
			}
			m, ok := s.taggedFields[tag]
			if !ok {
				s.taggedFields[tag] = make(map[string]*field)
				m = s.taggedFields[tag]
			}
			m[name] = fie
		}
	}
}

type setter interface {
	Add(string, string)
}

func (s *structConfig) encode(src reflect.Value, tag StructTag, dst setter) errx.Error {
	fields, ok := s.taggedFields[tag]
	if !ok || len(fields) == 0 {
		return nil
	}
	for k := range fields {
		f := fields[k]
		if f.codec == nil {
			logrus.Errorf("No converter found for field %s with tag %s", k, tag)
			continue
		}
		current := src
		for _, index := range f.index {
			v := current.Field(index)
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					continue
				}
				v = v.Elem()
			}
			current = v
		}
		if current.Kind() == reflect.Slice || current.Kind() == reflect.Array {
			for i := 0; i < current.Len(); i++ {
				v := current.Index(i)
				for v.Kind() == reflect.Ptr {
					if v.IsNil() {
						continue
					}
					v = v.Elem()
				}
				str, err := f.codec.ToString(v)
				if err != nil {
					return errx.Wrap(err).AppendMsgf("Failed to convert field %s[%d] to string", k, i).Err()
				}
				dst.Add(k, str)
			}
		} else {
			str, err := f.codec.ToString(current)
			if err != nil {
				return errx.Wrap(err).AppendMsgf("Failed to convert field %s to string", k).Err()
			}
			if str != "" || !f.omitempty {
				dst.Add(k, str)
			}
		}
	}
	return nil
}

type getter interface {
	Get(string) string
	Values(string) []string
}

func (s *structConfig) decode(src getter, tag StructTag, dst reflect.Value) errx.Error {
	fields, ok := s.taggedFields[tag]
	if !ok || len(fields) == 0 {
		return nil
	}
	if src == nil {
		return nil
	}
	for k := range fields {
		f := fields[k]
		if f.codec == nil {
			logrus.Errorf("No converter found for field %s with tag %s", k, tag)
			continue
		}
		value := src.Get(k)
		if value == "" {
			continue
		}
		current := dst
		for _, index := range f.index {
			v := current.Field(index)
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					v.Set(reflect.New(v.Type().Elem()))
				}
				v = v.Elem()
			}
			current = v
		}
		if current.Kind() == reflect.Slice || current.Kind() == reflect.Array {
			for i, value := range src.Values(k) {
				v := reflect.New(current.Type().Elem()).Elem()
				err := f.codec.FromString(v, value)
				if err != nil {
					return errx.Wrap(err).AppendMsgf("Failed to convert string '%s' to field %s[%d]", value, k, i).Err()
				}
				current.Set(reflect.Append(current, v))
			}
		} else {
			err := f.codec.FromString(current, value)
			if err != nil {
				return errx.Wrap(err).AppendMsgf("Failed to convert string '%s' to field %s", value, k).Err()
			}
		}
	}
	return nil
}

func (s *structConfig) excludeIntersection(tag StructTag) {
	fields, ok := s.taggedFields[tag]
	if !ok || len(fields) == 0 {
		return
	}
	for t, m := range s.taggedFields {
		if tag == t {
			continue
		}
		for k := range m {
			delete(fields, k)
		}
	}
	if len(fields) == 0 {
		delete(s.taggedFields, tag)
	}
}
