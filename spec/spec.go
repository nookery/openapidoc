// Package spec provides the OpenAPI 3.1 document model and JSON Schema
// generation from Go types.
//
// The document model is a deliberately small subset of OpenAPI 3.1 that
// covers everything needed to describe a JSON HTTP API. Schemas follow the
// JSON Schema 2020-12 dialect that OpenAPI 3.1 adopts.
package spec

// Document is an OpenAPI 3.1 document.
type Document struct {
	OpenAPI    string                `json:"openapi"`
	Info       Info                  `json:"info"`
	Servers    []Server              `json:"servers,omitempty"`
	Paths      map[string]PathItem   `json:"paths"`
	Components *Components           `json:"components,omitempty"`
	Security   []SecurityRequirement `json:"security,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty"`
}

// Info holds the document level metadata.
type Info struct {
	Title          string   `json:"title"`
	Version        string   `json:"version"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
}

// Contact is the API contact information.
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License is the API license information.
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Server describes an API server.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PathItem holds the operations available on a single path.
type PathItem struct {
	Get         *Operation `json:"get,omitempty"`
	Put         *Operation `json:"put,omitempty"`
	Post        *Operation `json:"post,omitempty"`
	Delete      *Operation `json:"delete,omitempty"`
	Patch       *Operation `json:"patch,omitempty"`
	Options     *Operation `json:"options,omitempty"`
	Head        *Operation `json:"head,omitempty"`
	Trace       *Operation `json:"trace,omitempty"`
	Summary     string     `json:"summary,omitempty"`
	Description string     `json:"description,omitempty"`
}

// Operation describes a single API operation on a path.
type Operation struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]*Response  `json:"responses,omitempty"`
	Security    []SecurityRequirement `json:"security,omitempty"`
	Deprecated  bool                  `json:"deprecated,omitempty"`
}

// Parameter describes a single operation parameter.
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
	Example     any     `json:"example,omitempty"`
}

// RequestBody describes the request payload of an operation.
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required,omitempty"`
}

// MediaType maps a content type to its schema.
type MediaType struct {
	Schema  *Schema `json:"schema,omitempty"`
	Example any     `json:"example,omitempty"`
}

// Response describes a single response of an operation.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Headers     map[string]Header    `json:"headers,omitempty"`
}

// Header describes a response header.
type Header struct {
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// Components holds the reusable objects of a document.
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme describes an authentication scheme.
type SecurityScheme struct {
	Type         string      `json:"type"`
	Description  string      `json:"description,omitempty"`
	Name         string      `json:"name,omitempty"`
	In           string      `json:"in,omitempty"`
	Scheme       string      `json:"scheme,omitempty"`
	BearerFormat string      `json:"bearerFormat,omitempty"`
	Flows        *OAuthFlows `json:"flows,omitempty"`
}

// OAuthFlows describes the supported OAuth flow variants.
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

// OAuthFlow describes a single OAuth flow.
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

// SecurityRequirement maps a security scheme name to the required scopes.
type SecurityRequirement map[string][]string

// Tag groups operations in the documentation UI.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
