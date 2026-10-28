// Package api provides a declarative way to describe HTTP operations.
//
// An operation is built from typed helpers that mirror the parts of an HTTP
// request and response. The values are collected by a Registry, which turns
// them into an OpenAPI document.
//
// The design goal is that a Go developer adds one line next to a route
// registration and gets a complete, accurate API description, with no code
// generation step and no build-time tooling.
package api

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/nookery/openapidoc/spec"
)

// Operation describes a single HTTP operation.
//
// The zero value is not useful; build one with Path, Query, Body and Resp.
//
// Schemas are not resolved when the operation is built. The Go types are
// kept as reflect.Type and resolved later by the Registry, so that all
// operations share one reflector and can reference each other's components.
type Operation struct {
	Method       string
	Path         string
	SummaryText  string
	Description  string
	OperationID  string
	Tags         []string
	Deprecated   bool
	Parameters   []Parameter
	RequestBody  *RequestBody
	Responses    map[string]*Response
	SecurityReqs []spec.SecurityRequirement
	OptionalSecs []spec.SecurityRequirement
}

// Parameter describes a single request parameter.
type Parameter struct {
	Name        string
	In          string
	Description string
	Required    bool
	Type        reflect.Type
	Schema      *spec.Schema
	Example     any
}

// RequestBody describes the payload of a request.
type RequestBody struct {
	Description string
	Required    bool
	ContentType string
	Type        reflect.Type
	Schema      *spec.Schema
}

// Response describes a single response.
type Response struct {
	// status is the map key of the response, such as "200 OK".
	status      string
	Description string
	ContentType string
	Type        reflect.Type
	Schema      *spec.Schema
	Headers     map[string]spec.Header
}

// Path returns a new operation for the given HTTP method and path.
//
// The path may contain {param} placeholders, for example "/users/{id}".
func Path(method, path string) *Operation {
	return &Operation{
		Method:    normalizeMethod(method),
		Path:      path,
		Responses: map[string]*Response{},
	}
}

// Get returns a new GET operation.
func Get(path string) *Operation { return Path(http.MethodGet, path) }

// Post returns a new POST operation.
func Post(path string) *Operation { return Path(http.MethodPost, path) }

// Put returns a new PUT operation.
func Put(path string) *Operation { return Path(http.MethodPut, path) }

// Patch returns a new PATCH operation.
func Patch(path string) *Operation { return Path(http.MethodPatch, path) }

// Delete returns a new DELETE operation.
func Delete(path string) *Operation { return Path(http.MethodDelete, path) }

// Summary sets the one line summary of the operation.
func (o *Operation) Summary(s string) *Operation {
	o.SummaryText = s
	return o
}

// Describe sets the long description of the operation.
func (o *Operation) Describe(s string) *Operation {
	o.Description = s
	return o
}

// ID sets the operation identifier, used to generate client method names.
func (o *Operation) ID(s string) *Operation {
	o.OperationID = s
	return o
}

// Tag adds a tag to the operation, grouping it in the documentation UI.
func (o *Operation) Tag(tags ...string) *Operation {
	o.Tags = append(o.Tags, tags...)
	return o
}

// MarkDeprecated marks the operation as deprecated.
func (o *Operation) MarkDeprecated() *Operation {
	o.Deprecated = true
	return o
}

// Query adds a query string parameter of the given Go type.
func (o *Operation) Query(name string, value any) *Operation {
	return o.param("query", name, value, false)
}

// RequiredQuery adds a required query string parameter.
func (o *Operation) RequiredQuery(name string, value any) *Operation {
	return o.param("query", name, value, true)
}

// Header adds a request header parameter.
func (o *Operation) Header(name string, value any) *Operation {
	return o.param("header", name, value, false)
}

// PathParam adds a path parameter. Path parameters are always required.
func (o *Operation) PathParam(name string, value any) *Operation {
	return o.param("path", name, value, true)
}

// Cookie adds a cookie parameter.
func (o *Operation) Cookie(name string, value any) *Operation {
	return o.param("cookie", name, value, false)
}

// param adds a parameter whose schema is derived from the Go type of value.
func (o *Operation) param(in, name string, value any, required bool) *Operation {
	o.Parameters = append(o.Parameters, Parameter{
		Name:     name,
		In:       in,
		Required: required,
		Type:     typeOf(value),
	})
	return o
}

// Param adds a parameter with an explicit schema.
func (o *Operation) Param(p Parameter) *Operation {
	o.Parameters = append(o.Parameters, p)
	return o
}

// Body sets the request payload, derived from the Go type of value.
//
// Passing a nil value produces a generic object schema.
func (o *Operation) Body(value any) *Operation {
	o.RequestBody = &RequestBody{
		Required:    true,
		ContentType: "application/json",
		Type:        typeOf(value),
	}
	return o
}

// OptionalBody sets the request payload and marks it as not required.
func (o *Operation) OptionalBody(value any) *Operation {
	o.RequestBody = &RequestBody{
		Required:    false,
		ContentType: "application/json",
		Type:        typeOf(value),
	}
	return o
}

// BodyWith sets the request payload with an explicit content type.
func (o *Operation) BodyWith(contentType string, value any) *Operation {
	o.RequestBody = &RequestBody{
		Required:    true,
		ContentType: contentType,
		Type:        typeOf(value),
	}
	return o
}

// Resp adds a response with the given status code, derived from the Go type
// of value.
//
// A nil value produces a response without a schema.
func (o *Operation) Resp(status int, value any) *Operation {
	return o.respWith(status, "application/json", value)
}

// RespWith adds a response with an explicit content type.
func (o *Operation) RespWith(status int, contentType string, value any) *Operation {
	return o.respWith(status, contentType, value)
}

// respWith implements RespWith.
func (o *Operation) respWith(status int, contentType string, value any) *Operation {
	if o.Responses == nil {
		o.Responses = map[string]*Response{}
	}
	r := &Response{
		status:      statusKey(status),
		Description: statusText(status),
		ContentType: contentType,
		Type:        typeOf(value),
	}
	o.Responses[r.status] = r
	return o
}

// RespHeader declares a response header on every response added so far.
func (o *Operation) RespHeader(name, description string, value any) *Operation {
	for _, r := range o.Responses {
		if r.Headers == nil {
			r.Headers = map[string]spec.Header{}
		}
		r.Headers[name] = spec.Header{
			Description: description,
			Schema:      &spec.Schema{Type: "string"},
		}
	}
	return o
}

// Security requires the given security schemes on this operation.
func (o *Operation) Security(names ...string) *Operation {
	o.SecurityReqs = append(o.SecurityReqs, newSecurityRequirement(names))
	return o
}

// SecurityOptional makes the given security schemes optional on this
// operation, meaning the endpoint may be called with or without credentials.
func (o *Operation) SecurityOptional(names ...string) *Operation {
	o.OptionalSecs = append(o.OptionalSecs, newSecurityRequirement(names))
	return o
}

// newSecurityRequirement builds a requirement entry for the given schemes.
func newSecurityRequirement(names []string) spec.SecurityRequirement {
	req := make(spec.SecurityRequirement, len(names))
	for _, n := range names {
		req[n] = []string{}
	}
	return req
}

// statusKey renders an HTTP status code as the map key used by OpenAPI.
//
// OpenAPI requires response keys to be the numeric status code, such as
// "200", or the literal "default".
func statusKey(status int) string {
	return strconv.Itoa(status)
}

// statusText returns the human readable name of a status code, falling
// back to the numeric code for codes the standard library does not know.
func statusText(status int) string {
	if text := http.StatusText(status); text != "" {
		return text
	}
	return strconv.Itoa(status)
}

// normalizeMethod upper cases an HTTP method and defaults to GET.
func normalizeMethod(method string) string {
	if method == "" {
		return http.MethodGet
	}
	return strings.ToUpper(method)
}

// typeOf returns the reflect.Type of a value, dereferencing pointers. It
// returns nil for a nil value.
func typeOf(value any) reflect.Type {
	if value == nil {
		return nil
	}
	t := reflect.TypeOf(value)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
