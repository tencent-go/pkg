package util

type StructTag string

const (
	TagHeader StructTag = "header"
	TagQuery  StructTag = "query"
	TagPath   StructTag = "path"
	TagJson   StructTag = "json"
	TagForm   StructTag = "form"
	TagEnv    StructTag = "env"
)

// func (d StructTag) Enum() Enum {
// 	return RegisterEnum(TagHeader, TagQuery, TagPath, TagJson, TagForm, TagEnv)
// }
