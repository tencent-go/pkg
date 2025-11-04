package schema

import (
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/tencent-go/pkg/types"
	"github.com/tencent-go/pkg/util"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type Collection interface {
	Packages() []*Package
	ParseAndGetType(typ reflect.Type, tag util.StructTag) (*Type, bool)
}

func NewCollection() Collection {
	return &collection{
		packageMap: make(map[string]*pkg),
		types:      make(map[string]*Type),
	}
}

type collection struct {
	packageMap map[string]*pkg
	types      map[string]*Type
}

func (c *collection) Packages() []*Package {
	names := make([]string, 0, len(c.packageMap))
	for name := range c.packageMap {
		names = append(names, name)
	}
	sort.Strings(names)
	res := make([]*Package, 0, len(names))
	for _, name := range names {
		res = append(res, c.packageMap[name].Package)
	}
	return res
}

func (c *collection) ParseAndGetType(typ reflect.Type, tag util.StructTag) (*Type, bool) {
	return c.parseAndGetType(typ, tag)
}

func (c *collection) parseAndGetType(typ reflect.Type, tag util.StructTag) (*Type, bool) {
	t := &Type{}

	for typ.Kind() == reflect.Ptr {
		t.Nullable = true
		typ = typ.Elem()
	}

	// base type
	t.BaseType = getBaseType(typ)

	switch t.BaseType {
	case BaseTypeArray:
		t.Nullable = true
		arrType, ok := c.parseAndGetType(typ.Elem(), tag)
		if !ok {
			return nil, false
		}
		t.Array = arrType
	case BaseTypeMap:
		keyType, ok := c.parseAndGetType(typ.Key(), tag)
		if !ok {
			logrus.Errorf("Unexpected map key type: %s", typ.String())
			return nil, false
		}
		valueType, ok := c.parseAndGetType(typ.Elem(), tag)
		if !ok {
			logrus.Errorf("Unexpected map value type: %s", typ.String())
			return nil, false
		}
		t.Nullable = true
		t.Map = &Map{ValueType: *valueType, KeyType: *keyType}
	case BaseTypeClass:
		cl, ok := c.parseAndGetClass(typ, tag)
		if !ok {
			return nil, false
		}
		t.Class = cl
	case BaseTypeString:
		if e, ok := c.parseAndGetEnum(typ, false); ok {
			t.Enum = e
		}
	case BaseTypeNumber:
		if e, ok := c.parseAndGetEnum(typ, true); ok {
			t.Enum = e
		}
	default:

	}
	return t, true
}

var (
	decimalType = reflect.TypeOf(decimal.Zero)
	idType      = reflect.TypeOf(types.EmptyID)
	anyType     = reflect.TypeOf((*interface{})(nil)).Elem()
)

func getBaseType(t reflect.Type) BaseType {
	//特殊类型
	if t == decimalType {
		return BaseTypeString
	}
	if t == idType {
		return BaseTypeString
	}
	if types.IsNilType(t) {
		return BaseTypeNull
	}
	if t == anyType {
		return BaseTypeAny
	}
	k := t.Kind()
	switch k {
	case reflect.Array, reflect.Slice:
		return BaseTypeArray
	case reflect.Map:
		return BaseTypeMap
	case reflect.Struct:
		return BaseTypeClass
	case reflect.String:
		return BaseTypeString
	case reflect.Bool:
		return BaseTypeBoolean
	default:
		if (k >= reflect.Int && k <= reflect.Uint64) || k == reflect.Float32 || k == reflect.Float64 {
			return BaseTypeNumber
		} else {
			logrus.Warnf("type unknown, kind: %d, pkgPath: %s", k, t.PkgPath())
			return BaseTypeAny
		}
	}
}

func (c *collection) getPkg(typ reflect.Type) *pkg {
	get := func(name string) (*pkg, bool) {
		p, ok := c.packageMap[name]
		if !ok {
			c.packageMap[name] = &pkg{
				Package: &Package{
					Name: name,
				},
				fullPath: typ.PkgPath(),
				classMap: make(map[string]*Class),
				enumMap:  make(map[string]*Enum),
			}
			return c.packageMap[name], true
		}
		if ok && p.fullPath == typ.PkgPath() {
			return p, true
		}
		return nil, false
	}
	if parts := strings.SplitN(typ.String(), ".", 2); len(parts) == 2 {
		name := parts[0]
		name = strings.ToUpper(name[:1]) + name[1:]
		if res, ok := get(name); ok {
			return res
		}
	} else {
		panic("TODO")
	}
	duplicate := map[string]bool{}
	var names []string
	for _, part := range pkgNameRe.Split(typ.PkgPath(), -1) {
		if duplicate[strings.ToLower(part)] {
			continue
		}
		duplicate[strings.ToLower(part)] = true
		part = strings.ToUpper(part[:1]) + part[1:]
		names = append(names, part)
	}
	for i := range len(names) - 1 {
		start := len(names) + i
		name := strings.Join(names[start:], "")
		if res, ok := get(name); ok {
			return res
		}
	}
	panic("TODO")
}

var genericRe = regexp.MustCompile(`\[(.+?)\]$`)

//var genericPlaceholders = []string{"T", "U", "V", "W", "X", "Y", "Z"}

func (c *collection) parseAndGetClass(typ reflect.Type, tag util.StructTag) (*Class, bool) {
	if !hasTaggedField(typ, tag) {
		return nil, false
	}
	p := c.getPkg(typ)
	name := typ.Name()
	//var generics []string
	if g := genericRe.FindString(name); g != "" {
		logrus.Panicf("unexpected generic class name: %s", name)
		name = name[:len(name)-len(g)]
	}

	// 匿名結構體
	if name == "" {
		logrus.Panicf("unsupported generic class type: %s", typ.String())
	}
	cls, exists := p.classMap[name]
	if exists {
		return cls, true
	}
	if tag != util.TagJson {
		suffix := strings.ToUpper(string(tag[:1])) + string(tag[1:])
		name = name + suffix
	}
	class, ok := p.classMap[name]
	if ok {
		return class, true
	}
	class = &Class{
		GoType:  typ,
		Name:    name,
		Package: p.Package,
		Tag:     tag,
		//Generics: generics,
		Fields: make([]Field, 0, typ.NumField()),
	}
	p.classMap[name] = class
	p.Classes = append(p.Classes, class)
	c.parseFields(typ, tag, &class.Fields)
	return class, true
}

func (c *collection) parseFields(typ reflect.Type, _tag util.StructTag, fields *[]Field) {
	for i := 0; i < typ.NumField(); i++ {
		goField := typ.Field(i)
		if goField.Anonymous {
			c.parseFields(goField.Type, _tag, fields)
			continue
		}

		t, ok := c.parseAndGetType(goField.Type, _tag)
		if !ok {
			continue
		}

		tag, ok := goField.Tag.Lookup(string(_tag))
		if !ok || tag == "-" {
			continue
		}
		name := goField.Name
		var optional bool
		if tag != "" {
			parts := strings.SplitN(tag, ",", 2)
			name = parts[0]
			if len(parts) > 1 {
				if parts[1] == "omitempty" {
					optional = true
				}
			}
		}
		f := Field{
			GoField:  goField,
			Name:     name,
			Type:     *t,
			Optional: optional,
		}
		*fields = append(*fields, f)
	}
}

func hasTaggedField(typ reflect.Type, tag util.StructTag) bool {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue
		}
		if f.Anonymous {
			if hasTaggedField(f.Type, tag) {
				return true
			}
		}
		if !f.IsExported() {
			continue
		}
		t, ok := f.Tag.Lookup(string(tag))
		if !ok || t == "-" {
			continue
		}
		return true
	}
	return false
}

var pkgNameRe = regexp.MustCompile(`[_\-/]+`)

var enumType = reflect.TypeOf((*types.IEnum)(nil)).Elem()

func (c *collection) parseAndGetEnum(t reflect.Type, isNumeric bool) (*Enum, bool) {
	if !t.Implements(enumType) {
		return nil, false
	}
	items := reflect.New(t).Elem().Interface().(types.IEnum).Enum().Items()
	if len(items) == 0 {
		return nil, false
	}
	p := c.getPkg(t)
	name := t.Name()
	if !strings.HasSuffix(name, "Enum") {
		name = name + "Enum"
	}
	e, exists := p.enumMap[name]
	if exists {
		return e, true
	}
	e = &Enum{
		GoType:    t,
		Name:      name,
		Package:   p.Package,
		Items:     items,
		IsNumeric: isNumeric,
	}
	p.enumMap[name] = e
	p.Enums = append(p.Enums, e)
	return e, true
}

type pkg struct {
	*Package
	fullPath string
	classMap map[string]*Class
	enumMap  map[string]*Enum
}
