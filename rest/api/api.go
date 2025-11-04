package api

import (
	"reflect"
)

type Node interface {
	Name() *string
	Path() *string
	Description() *string
	RequestContentType() *ContentType
	ResponseContentType() *ContentType
	RequireAuthentication() *bool
	RequireAuthorization() *bool
	RequireWrapOutput() *bool
}

type Endpoint interface {
	Node
	Method() Method
	InputType() reflect.Type
	OutputType() reflect.Type
}

type EndpointBuilder[I, O any] interface {
	Endpoint
	WithName(name string) EndpointBuilder[I, O]
	WithPath(path string) EndpointBuilder[I, O]
	WithMethod(method Method) EndpointBuilder[I, O]
	WithDescription(description string) EndpointBuilder[I, O]
	WithRequestContentType(contentType ContentType) EndpointBuilder[I, O]
	WithResponseContentType(contentType ContentType) EndpointBuilder[I, O]
	WithRequireAuthentication(required bool) EndpointBuilder[I, O]
	WithRequireAuthorization(required bool) EndpointBuilder[I, O]
	WithRequireWrapOutput(required bool) EndpointBuilder[I, O]
}

type Group interface {
	Node
	Match(method Method, path string) (MatchedRoute, bool)
	Routes() []Route
	Children() []Node
}

type GroupBuilder interface {
	Group
	WithName(name string) GroupBuilder
	WithPath(path string) GroupBuilder
	WithDescription(description string) GroupBuilder
	WithRequestContentType(contentType ContentType) GroupBuilder
	WithResponseContentType(contentType ContentType) GroupBuilder
	WithRequireAuthentication(required bool) GroupBuilder
	WithRequireAuthorization(required bool) GroupBuilder
	WithRequireWrapOutput(required bool) GroupBuilder
	WithChildren(children ...Node) GroupBuilder
}

type MatchedRoute interface {
	Route
	PathParams() map[string]string
}

type Route interface {
	Endpoint() Endpoint
	Path() string
	PathChain() []string
	RequestContentType() ContentType
	ResponseContentType() ContentType
	RequireAuthentication() bool
	RequireAuthorization() bool
	RequireWrapOutput() bool
	Ancestors() []Node
}

func NewEndpoint[I, O any]() EndpointBuilder[I, O] {
	return &endpoint[I, O]{}
}

func NewGroup() GroupBuilder {
	return &group{}
}
