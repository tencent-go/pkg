package tsdoc

import (
	"fmt"

	"github.com/tencent-go/pkg/doc/schema"
)

func parseType(t schema.Type, currentPkg *schema.Package) string {
	res := "unknown"
	//enum
	if t.Enum != nil {
		res = t.Enum.Name
		if t.Enum.Package != nil && t.Enum.Package != currentPkg {
			res = t.Enum.Package.Name + "." + res
		}
	} else {
		switch t.BaseType {
		case schema.BaseTypeArray:
			if t.Array != nil {
				res = parseType(*t.Array, currentPkg) + "[]"
			}
		case schema.BaseTypeMap:
			if t.Map != nil {
				res = fmt.Sprintf("Record<%s,%s>", parseType(t.Map.KeyType, currentPkg), parseType(t.Map.ValueType, currentPkg))
			}
		case schema.BaseTypeClass:
			if t.Class != nil {
				res = t.Class.Name
				if t.Class.Package != nil && t.Class.Package != currentPkg {
					res = t.Class.Package.Name + "." + res
				}
			}
		case schema.BaseTypeString:
			res = "string"
		case schema.BaseTypeNumber:
			res = "number"
		case schema.BaseTypeBoolean:
			res = "boolean"
		case schema.BaseTypeNull:
			res = "null"
		default:
			res = "any"
		}
	}
	if t.Nullable && res != "null" {
		res += " | null"
	}
	return res
}
