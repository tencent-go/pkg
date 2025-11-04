package schema

import (
	"github.com/tencent-go/pkg/types"
	"github.com/tencent-go/pkg/util"
	"reflect"
)

type Package struct {
	Name    string
	Classes []*Class
	Enums   []*Enum
}

type Class struct {
	GoType reflect.Type
	Name   string
	//Generics []string
	Package *Package
	Fields  []Field
	Tag     util.StructTag
}

type Field struct {
	GoField reflect.StructField
	Name    string
	//Generic  string
	Type     Type
	Optional bool
}

type Enum struct {
	GoType    reflect.Type
	Name      string
	Package   *Package
	Items     []types.EnumElement
	IsNumeric bool
}

type Map struct {
	GoType    reflect.Type
	KeyType   Type
	ValueType Type
}

type Type struct {
	GoType   reflect.Type
	BaseType BaseType
	Nullable bool
	Array    *Type
	Class    *Class
	//GenericType *Type
	Enum *Enum
	Map  *Map
}

type BaseType int

const (
	BaseTypeNull BaseType = iota
	BaseTypeAny
	BaseTypeNumber
	BaseTypeString
	BaseTypeBoolean
	BaseTypeClass
	BaseTypeArray
	BaseTypeMap
)
