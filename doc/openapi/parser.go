package openapi

import (
	"fmt"
	"path"
	"strings"

	"github.com/tencent-go/pkg/doc/restdoc"
	"github.com/tencent-go/pkg/doc/schema"
	"github.com/tencent-go/pkg/rest/api"
	"github.com/tencent-go/pkg/util"

	"github.com/tencent-go/pkg/types"
)

func NewDefault() *OpenAPI {
	return &OpenAPI{
		OpenAPI: "3.0.1",
		Info: Info{
			Title:   "API Documentation",
			Version: "1.0.0",
		},
		Paths: make(map[string]*PathItem),
		Components: &Components{
			SecuritySchemes: map[string]*SecurityScheme{
				"BearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
			},
			Schemas:    make(map[string]*Schema),
			Parameters: map[string]*Parameter{},
		},
		Security: []map[string][]string{{"BearerAuth": []string{}}},
	}
}

func (spec *OpenAPI) Parse(groups []restdoc.Group) *OpenAPI {
	for _, group := range groups {
		tag := Tag{
			Name:        group.Name,
			Description: group.Description,
		}
		spec.Tags = append(spec.Tags, tag)
		for _, item := range group.Endpoints {
			p := path.Join("/", item.Path)
			pi, ok := spec.Paths[p]
			if !ok {
				pi = &PathItem{}
				spec.Paths[p] = pi
			}
			o := spec.endpoint2Operation(item)
			o.Tags = []string{group.Name}
			switch item.Method {
			case api.MethodGet:
				pi.Get = o
			case api.MethodPost:
				pi.Post = o
			case api.MethodPut:
				pi.Put = o
			case api.MethodPatch:
				pi.Patch = o
			case api.MethodDelete:
				pi.Delete = o
			}
		}
	}
	return spec
}

func (spec *OpenAPI) endpoint2Operation(endpoint restdoc.Endpoint) *Operation {
	o := &Operation{
		//Security: []map[string][]string{{"BearerAuth": []string{}}},
	}
	var res *Schema
	//TODO: implement wrapper
	// if srv.WrapperType != nil {
	// 	res = &Schema{}
	// 	wrapper := &Schema{Ref: fmt.Sprintf("#/components/schemas/%s.%s", srv.WrapperType.PackageName, srv.WrapperType.Name)}
	// 	res.AllOf = append(res.AllOf, wrapper)
	// 	if srv.Response != nil {
	// 		data := &Schema{
	// 			Properties: map[string]*Schema{
	// 				srv.WrapperDataField: type2Schema(*srv.Response),
	// 			},
	// 		}
	// 		res.AllOf = append(res.AllOf, data)
	// 	}
	// } else {

	// }
	if endpoint.Response != nil {
		res = spec.type2Schema(*endpoint.Response)
	}
	if res != nil {
		o.Responses = map[string]Response{
			"200": {
				Content: map[string]MediaType{
					"application/json": {
						Schema: res,
					},
				},
			},
		}
	}

	if endpoint.Body != nil {
		o.RequestBody = &RequestBody{
			Content: map[string]MediaType{
				"application/json": {
					Schema: spec.type2Schema(*endpoint.Body),
				},
			},
		}
	}
	if endpoint.Query != nil {
		o.Parameters = append(o.Parameters, spec.class2parameters(*endpoint.Query)...)
	}
	if endpoint.Header != nil {
		o.Parameters = append(o.Parameters, spec.class2parameters(*endpoint.Header)...)
	}
	if endpoint.Param != nil {
		o.Parameters = append(o.Parameters, spec.class2parameters(*endpoint.Param)...)
	}

	var descriptions []string
	if !endpoint.AuthenticationRequired {
		descriptions = append(descriptions, "No authentication required")
	} else {
		descriptions = append(descriptions, "Authentication required")
		if !endpoint.AuthorizationRequired {
			descriptions = append(descriptions, "No authorization required")
		} else {
			descriptions = append(descriptions, fmt.Sprintf("Authorization required: %s", endpoint.Permission))
		}
	}
	o.Description = strings.Join(descriptions, "; ")
	summaries := []string{endpoint.Name}
	if endpoint.Description != "" {
		summaries = append(summaries, endpoint.Description)
	}
	o.Summary = strings.Join(summaries, " ")
	return o
}

func (spec *OpenAPI) class2parameters(class schema.Class) []Parameter {
	var parameters []Parameter
	for _, field := range class.Fields {
		p := Parameter{
			Name:     field.Name,
			Required: !field.Optional,
			Schema:   spec.type2Schema(field.Type),
		}
		switch class.Tag {
		case util.TagQuery:
			p.In = "query"
		case util.TagPath:
			p.In = "path"
		case util.TagHeader:
			p.In = "header"
		default:
			continue
		}
		parameters = append(parameters, p)
	}
	return parameters
}

func (spec *OpenAPI) class2Schema(class schema.Class) *Schema {
	s := &Schema{
		Types:      []string{"object"},
		Properties: make(map[string]*Schema),
	}
	for _, field := range class.Fields {
		//if !field.Optional {
		//	s.Required = append(s.Required, field.Name)
		//}
		s.Properties[field.Name] = spec.type2Schema(field.Type)
	}
	return s
}

func (spec *OpenAPI) type2Schema(t schema.Type) *Schema {
	s := &Schema{}
	if t.Nullable {
		s.Nullable = true
	}
	if t.Enum != nil {
		key := t.Enum.Package.Name + "." + t.Enum.Name
		s.Ref = "#/components/schemas/" + key
		if _, ok := spec.Components.Schemas[key]; !ok {
			spec.Components.Schemas[key] = spec.enum2Schema(*t.Enum)
		}
		return s
	}
	switch t.BaseType {
	case schema.BaseTypeMap:
		s.Types = []string{"object"}
		s.AdditionalProperties = spec.type2Schema(t.Map.ValueType)
	case schema.BaseTypeArray:
		s.Types = []string{"array"}
		s.Items = spec.type2Schema(*t.Array)
	case schema.BaseTypeClass:
		key := t.Class.Package.Name + "." + t.Class.Name
		s.Ref = "#/components/schemas/" + key
		if _, ok := spec.Components.Schemas[key]; !ok {
			spec.Components.Schemas[key] = spec.class2Schema(*t.Class)
		}
	case schema.BaseTypeString:
		s.Types = []string{"string"}
	case schema.BaseTypeNumber:
		s.Types = []string{"number"}
	case schema.BaseTypeBoolean:
		s.Types = []string{"boolean"}
	case schema.BaseTypeNull:
		s.Nullable = true
		s.Types = []string{"object", "null"}
		//fallthrough
	default:
		s.Types = []string{"object"}
	}
	return s
}

func (spec *OpenAPI) enum2Schema(enum schema.Enum) *Schema {
	s := &Schema{
		Properties: make(map[string]*Schema),
	}
	if enum.IsNumeric {
		s.Types = []string{"integer"}
	} else {
		s.Types = []string{"string"}
	}
	var descriptions []string
	for _, item := range enum.Items {
		s.Enum = append(s.Enum, item.Value)
		if len(item.LocalizedInfo) > 0 {
			d := fmt.Sprintf("%v: %s", item.Value, item.LocalizedInfo.Get(types.DefaultLocale).Label)
			descriptions = append(descriptions, d)
		}
	}
	if len(descriptions) > 0 {
		s.Description = strings.Join(descriptions, ", ")
	}
	return s
}
