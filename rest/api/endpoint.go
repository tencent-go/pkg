package api

import (
	"reflect"
	"strings"
)

type node struct {
	name                  *string
	path                  *string
	description           *string
	requestContentType    *ContentType
	responseContentType   *ContentType
	requireAuthentication *bool
	requireAuthorization  *bool
	requireWrapOutput     *bool
	parent                Node
}

func (n *node) Name() *string {
	return n.name
}

func (n *node) Path() *string {
	return n.path
}

func (n *node) Description() *string {
	return n.description
}

func (n *node) RequestContentType() *ContentType {
	return n.requestContentType
}

func (n *node) ResponseContentType() *ContentType {
	return n.responseContentType
}

func (n *node) RequireAuthentication() *bool {
	return n.requireAuthentication
}

func (n *node) RequireAuthorization() *bool {
	return n.requireAuthorization
}

func (n *node) RequireWrapOutput() *bool {
	return n.requireWrapOutput
}

type endpoint[I, O any] struct {
	node
	method Method
}

func (a *endpoint[I, O]) copy() *endpoint[I, O] {
	c := *a
	return &c
}

func (a *endpoint[I, O]) Method() Method {
	if a.method == "" {
		return MethodGet
	}
	return a.method
}

func (a *endpoint[I, O]) Name() *string {
	if a.name != nil {
		return a.name
	}
	var name, path string
	if a.path != nil {
		path = *a.path
	}
	var action string
	switch a.Method() {
	case MethodPost:
		action = "create"
	case MethodPut:
		action = "update"
	case MethodPatch:
		action = "update-partial"
	case MethodDelete:
		action = "remove"
	default:
		if strings.Contains(path, "{") {
			name = "get"
		} else {
			name = "list"
		}
		return &name
	}
	name = action
	if path != "" {
		fields := strings.FieldsFunc(path, func(r rune) bool {
			return r == '-' || r == '_' || r == '/' || r == '.'
		})
		parts := []string{action}
		for _, f := range fields {
			if f == "" {
				continue
			}
			if strings.HasPrefix(f, "{") {
				continue
			}
			parts = append(parts, strings.ToLower(f))
		}
		name = strings.Join(parts, "-")
	}
	return &name
}

func (a *endpoint[I, O]) InputType() reflect.Type {
	var ptr *I
	t := reflect.TypeOf(ptr)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func (a *endpoint[I, O]) OutputType() reflect.Type {
	var ptr *O
	t := reflect.TypeOf(ptr)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func (a *endpoint[I, O]) WithName(name string) EndpointBuilder[I, O] {
	c := a.copy()
	c.name = &name
	return c
}

func (a *endpoint[I, O]) WithPath(path string) EndpointBuilder[I, O] {
	c := a.copy()
	c.path = &path
	return c
}

func (a *endpoint[I, O]) WithMethod(method Method) EndpointBuilder[I, O] {
	c := a.copy()
	c.method = method
	return c
}

func (a *endpoint[I, O]) WithDescription(description string) EndpointBuilder[I, O] {
	c := a.copy()
	c.description = &description
	return c
}

func (a *endpoint[I, O]) WithRequestContentType(contentType ContentType) EndpointBuilder[I, O] {
	c := a.copy()
	c.requestContentType = &contentType
	return c
}

func (a *endpoint[I, O]) WithResponseContentType(contentType ContentType) EndpointBuilder[I, O] {
	c := a.copy()
	c.responseContentType = &contentType
	return c
}

func (a *endpoint[I, O]) WithRequireAuthentication(required bool) EndpointBuilder[I, O] {
	c := a.copy()
	c.requireAuthentication = &required
	return c
}

func (a *endpoint[I, O]) WithRequireAuthorization(required bool) EndpointBuilder[I, O] {
	c := a.copy()
	c.requireAuthorization = &required
	return c
}

func (a *endpoint[I, O]) WithRequireWrapOutput(required bool) EndpointBuilder[I, O] {
	c := a.copy()
	c.requireWrapOutput = &required
	return c
}
