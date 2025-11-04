package openapi

// OpenAPI represents the root of an OpenAPI 3.0 specification
type OpenAPI struct {
	OpenAPI      string                 `json:"openapi"` // Version of OpenAPI spec
	Info         Info                   `json:"info"`
	Paths        map[string]*PathItem   `json:"paths"`
	Components   *Components            `json:"components,omitempty"`
	Servers      []Server               `json:"servers,omitempty"`
	Tags         []Tag                  `json:"tags,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
	Security     []map[string][]string  `json:"security,omitempty"`
}

// Info represents metadata about the API
type Info struct {
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
	Version        string   `json:"version"`
}

// Contact represents the contact information for the API
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License represents the license information for the API
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Server represents a server hosting the API
type Server struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty"`
}

// ServerVariable represents a server variable for URL template substitution
type ServerVariable struct {
	Default     string   `json:"default"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// PathItem describes the operations available on a single path
type PathItem struct {
	Summary     string     `json:"summary,omitempty"`
	Description string     `json:"description,omitempty"`
	Get         *Operation `json:"get,omitempty"`
	Put         *Operation `json:"put,omitempty"`
	Post        *Operation `json:"post,omitempty"`
	Delete      *Operation `json:"delete,omitempty"`
	Options     *Operation `json:"options,omitempty"`
	Head        *Operation `json:"head,omitempty"`
	Patch       *Operation `json:"patch,omitempty"`
	Trace       *Operation `json:"trace,omitempty"`
}

// Operation describes a single API operation on a path
type Operation struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Deprecated  bool                  `json:"deprecated,omitempty"`
	Security    []map[string][]string `json:"security,omitempty"`
	Servers     []Server              `json:"servers,omitempty"`
}

// Parameter describes a single operation parameter
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // Query, header, path, or cookie
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody describes a single request body
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required,omitempty"`
}

// MediaType represents a media type for request/response bodies
type MediaType struct {
	Schema   *Schema            `json:"schema,omitempty"`
	Examples map[string]Example `json:"examples,omitempty"`
}

// Response describes a single response from an API operation
type Response struct {
	Description string               `json:"description"`
	Headers     map[string]Header    `json:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// Header represents a single header in a response
type Header struct {
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// Components holds a set of reusable objects
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	Responses       map[string]*Response       `json:"responses,omitempty"`
	Parameters      map[string]*Parameter      `json:"parameters,omitempty"`
	Examples        map[string]*Example        `json:"examples,omitempty"`
	RequestBodies   map[string]*RequestBody    `json:"requestBodies,omitempty"`
	Headers         map[string]*Header         `json:"headers,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
}

// Schema represents a JSON Schema object
type Schema struct {
	Type                 string                 `json:"type,omitempty"`
	Types                []string               `json:"types,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Properties           map[string]*Schema     `json:"properties,omitempty"`
	Items                *Schema                `json:"items,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Nullable             bool                   `json:"nullable,omitempty"`
	Ref                  string                 `json:"$ref,omitempty"`
	Enum                 []any                  `json:"enum,omitempty"`
	AdditionalProperties *Schema                `json:"additionalProperties,omitempty"`
	AllOf                []*Schema              `json:"allOf,omitempty"`
	OneOf                []*Schema              `json:"oneOf,omitempty"`
	AnyOf                []*Schema              `json:"anyOf,omitempty"`
	Not                  *Schema                `json:"not,omitempty"`
	Discriminator        *Discriminator         `json:"discriminator,omitempty"`
	Title                string                 `json:"title,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Default              interface{}            `json:"default,omitempty"`
	Example              interface{}            `json:"example,omitempty"`
	Examples             map[string]interface{} `json:"examples,omitempty"`
	Deprecated           bool                   `json:"deprecated,omitempty"`
	Maximum              *float64               `json:"maximum,omitempty"`
	ExclusiveMaximum     bool                   `json:"exclusiveMaximum,omitempty"`
	Minimum              *float64               `json:"minimum,omitempty"`
	ExclusiveMinimum     bool                   `json:"exclusiveMinimum,omitempty"`
	MaxLength            *int                   `json:"maxLength,omitempty"`
	MinLength            *int                   `json:"minLength,omitempty"`
	Pattern              string                 `json:"pattern,omitempty"`
	MaxItems             *int                   `json:"maxItems,omitempty"`
	MinItems             *int                   `json:"minItems,omitempty"`
	UniqueItems          bool                   `json:"uniqueItems,omitempty"`
	MaxProperties        *int                   `json:"maxProperties,omitempty"`
	MinProperties        *int                   `json:"minProperties,omitempty"`
	MultipleOf           *float64               `json:"multipleOf,omitempty"`
	ReadOnly             bool                   `json:"readOnly,omitempty"`
	WriteOnly            bool                   `json:"writeOnly,omitempty"`
	ExternalDocs         *ExternalDocs          `json:"externalDocs,omitempty"`
	Xml                  *XML                   `json:"xml,omitempty"`
}

type Discriminator struct {
	PropertyName string            `json:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty"`
}

type ExternalDocs struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

type XML struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
	Attribute bool   `json:"attribute,omitempty"`
	Wrapped   bool   `json:"wrapped,omitempty"`
}

// Example represents an example object
type Example struct {
	Summary     string      `json:"summary,omitempty"`
	Description string      `json:"description,omitempty"`
	Value       interface{} `json:"value,omitempty"`
}

// SecurityScheme represents a security scheme that can be used by the API
type SecurityScheme struct {
	Type             string      `json:"type"`
	Description      string      `json:"description,omitempty"`
	Name             string      `json:"name,omitempty"`
	In               string      `json:"in,omitempty"`
	Scheme           string      `json:"scheme,omitempty"`
	BearerFormat     string      `json:"bearerFormat,omitempty"`
	Flows            *OAuthFlows `json:"flows,omitempty"`
	OpenIDConnectURL string      `json:"openIdConnectUrl,omitempty"`
}

// OAuthFlows represents OAuth flow configuration
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

// OAuthFlow represents a single OAuth flow
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

// Tag adds metadata to a single tag used by the API
type Tag struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

// ExternalDocumentation allows referencing external resources
type ExternalDocumentation struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}
