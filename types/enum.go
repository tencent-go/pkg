package types

import (
	"reflect"

	"github.com/tencent-go/pkg/util"
)

type IEnum interface {
	Enum() Enum
}

type Enum interface {
	Items() []EnumElement
	Contains(val any) bool
}

type EnumElement struct {
	Value         any                              `json:"value"`
	LocalizedInfo LocalizedValues[EnumElementInfo] `json:"localizedInfo,omitempty"`
}

type EnumElementInfo struct {
	Label string `json:"label"`
	Tip   string `json:"tip,omitempty"`
}

type supportedTypes interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~string
}

type enumImpl struct {
	items  []EnumElement
	keyMap map[any]*EnumElement
}

func (d *enumImpl) Items() []EnumElement {
	return d.items
}

func (d *enumImpl) Contains(val any) bool {
	_, ok := d.keyMap[val]
	return ok
}

func newEnum[T supportedTypes](values ...T) *enumImpl {
	e := &enumImpl{
		items:  make([]EnumElement, len(values)),
		keyMap: make(map[any]*EnumElement),
	}
	for i, value := range values {
		e.items[i] = EnumElement{Value: value}
		e.keyMap[value] = &e.items[i]
	}
	return e
}

var store util.LazyMap[reflect.Type, *enumImpl]

func RegisterEnum[T supportedTypes](items ...T) Enum {
	if len(items) == 0 {
		panic("enum.Values: no items")
	}
	v, _ := store.LoadOrLazyStore(reflect.TypeOf(items[0]), func() *enumImpl {
		return newEnum(items...)
	})
	return v
}

type EnumExtender[T IEnum] interface {
	Value(value T) EnumExtender[T]
	Locale(Locale) EnumExtender[T]
	Label(label string) EnumExtender[T]
	Tip(tip string) EnumExtender[T]
}
type enumExtender[T IEnum] struct {
	locale Locale
	value  T
	key    string
	enum   *enumImpl
}

func (d *enumExtender[T]) Value(value T) EnumExtender[T] {
	d.value = value
	return d
}

func (d *enumExtender[T]) Locale(locale Locale) EnumExtender[T] {
	d.locale = locale
	return d
}

func (d *enumExtender[T]) Label(label string) EnumExtender[T] {
	item := d.enum.keyMap[d.value]
	if item.LocalizedInfo == nil {
		item.LocalizedInfo = LocalizedValues[EnumElementInfo]{}
	}
	info, ok := item.LocalizedInfo[d.locale]
	if !ok {
		info = EnumElementInfo{}
	}
	info.Label = label
	item.LocalizedInfo[d.locale] = info
	return d
}

func (d *enumExtender[T]) Tip(tip string) EnumExtender[T] {
	item := d.enum.keyMap[d.value]
	if item.LocalizedInfo == nil {
		item.LocalizedInfo = LocalizedValues[EnumElementInfo]{}
	}
	info, ok := item.LocalizedInfo[d.locale]
	if !ok {
		info = EnumElementInfo{}
	}
	info.Tip = tip
	item.LocalizedInfo[d.locale] = info
	return d
}

func ExtendEnum[T IEnum](anyElement T) EnumExtender[T] {
	e := anyElement.Enum()
	d, ok := e.(*enumImpl)
	if !ok {
		return nil
	}
	return &enumExtender[T]{
		enum:  d,
		value: anyElement,
	}
}
