package api

import (
	"sort"
	"strings"
	"sync"

	"github.com/nookery/openapidoc/spec"
)

// Registry collects operations and turns them into an OpenAPI document.
//
// A Registry is safe for concurrent use, so operations can be registered
// from separate goroutines during start up.
type Registry struct {
	mu         sync.Mutex
	operations []*Operation
	title      string
	version    string
	desc       string
	servers    []spec.Server
	tags       []spec.Tag
	schemes    map[string]*spec.SecurityScheme
	reflector  *spec.Reflector
}

// NewRegistry creates a registry with the given document title and version.
func NewRegistry(title, version string) *Registry {
	return &Registry{
		title:     title,
		version:   version,
		schemes:   map[string]*spec.SecurityScheme{},
		reflector: spec.NewReflector(),
	}
}

// Title sets the document title.
func (r *Registry) Title(t string) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.title = t
	return r
}

// Version sets the document version.
func (r *Registry) Version(v string) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.version = v
	return r
}

// Description sets the document description.
func (r *Registry) Description(d string) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.desc = d
	return r
}

// AddServer registers an API server.
func (r *Registry) AddServer(url, description string) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers = append(r.servers, spec.Server{URL: url, Description: description})
	return r
}

// AddTag registers a tag with an optional description.
func (r *Registry) AddTag(name, description string) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tags = append(r.tags, spec.Tag{Name: name, Description: description})
	return r
}

// AddSecurityScheme registers an authentication scheme that operations can
// refer to by name.
func (r *Registry) AddSecurityScheme(name string, scheme *spec.SecurityScheme) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schemes[name] = scheme
	return r
}

// BearerAuth registers a Bearer token security scheme and returns its name.
func (r *Registry) BearerAuth(name, description string) string {
	r.AddSecurityScheme(name, &spec.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  description,
	})
	return name
}

// APIKeyAuth registers an API key security scheme and returns its name.
func (r *Registry) APIKeyAuth(name, header, description string) string {
	r.AddSecurityScheme(name, &spec.SecurityScheme{
		Type:        "apiKey",
		Name:        header,
		In:          "header",
		Description: description,
	})
	return name
}

// Register adds an operation to the registry.
func (r *Registry) Register(ops ...*Operation) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, op := range ops {
		if op != nil {
			r.operations = append(r.operations, op)
		}
	}
	return r
}

// Operations returns the registered operations.
func (r *Registry) Operations() []*Operation {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Operation, len(r.operations))
	copy(out, r.operations)
	return out
}

// Document builds the OpenAPI document from the registered operations.
func (r *Registry) Document() *spec.Document {
	r.mu.Lock()
	defer r.mu.Unlock()

	doc := &spec.Document{
		OpenAPI: "3.1.0",
		Info: spec.Info{
			Title:       r.title,
			Version:     r.version,
			Description: r.desc,
		},
		Servers: r.servers,
		Paths:   map[string]spec.PathItem{},
		Tags:    r.tags,
	}

	for _, op := range r.operations {
		item, ok := doc.Paths[op.Path]
		if !ok {
			item = spec.PathItem{}
		}
		converted := r.convertOperation(op)
		switch op.Method {
		case "GET":
			item.Get = converted
		case "POST":
			item.Post = converted
		case "PUT":
			item.Put = converted
		case "DELETE":
			item.Delete = converted
		case "PATCH":
			item.Patch = converted
		case "OPTIONS":
			item.Options = converted
		case "HEAD":
			item.Head = converted
		case "TRACE":
			item.Trace = converted
		}
		doc.Paths[op.Path] = item
	}

	if len(r.schemes) > 0 {
		doc.Components = &spec.Components{
			SecuritySchemes: map[string]*spec.SecurityScheme{},
		}
		for name, scheme := range r.schemes {
			doc.Components.SecuritySchemes[name] = scheme
		}
	}

	if schemas := r.reflector.Schemas(); len(schemas) > 0 {
		if doc.Components == nil {
			doc.Components = &spec.Components{}
		}
		doc.Components.Schemas = schemas
	}

	return doc
}

// convertOperation turns a declared operation into its OpenAPI form.
func (r *Registry) convertOperation(op *Operation) *spec.Operation {
	out := &spec.Operation{
		Tags:        op.Tags,
		Summary:     op.SummaryText,
		Description: op.Description,
		OperationID: op.OperationID,
		Deprecated:  op.Deprecated,
		Responses:   map[string]*spec.Response{},
	}

	for _, p := range op.Parameters {
		schema := p.Schema
		if schema == nil && p.Type != nil {
			schema = r.reflector.SchemaOf(p.Type)
		}
		if schema == nil {
			schema = &spec.Schema{}
		}
		out.Parameters = append(out.Parameters, spec.Parameter{
			Name:        p.Name,
			In:          p.In,
			Description: p.Description,
			Required:    p.Required,
			Schema:      schema,
			Example:     p.Example,
		})
	}

	if op.RequestBody != nil {
		schema := op.RequestBody.Schema
		if schema == nil && op.RequestBody.Type != nil {
			schema = r.reflector.SchemaOf(op.RequestBody.Type)
		}
		if schema == nil {
			schema = &spec.Schema{}
		}
		contentType := op.RequestBody.ContentType
		if contentType == "" {
			contentType = "application/json"
		}
		out.RequestBody = &spec.RequestBody{
			Description: op.RequestBody.Description,
			Required:    op.RequestBody.Required,
			Content: map[string]spec.MediaType{
				contentType: {Schema: schema},
			},
		}
	}

	for _, resp := range sortedResponses(op.Responses) {
		schema := resp.Schema
		if schema == nil && resp.Type != nil {
			schema = r.reflector.SchemaOf(resp.Type)
		}
		contentType := resp.ContentType
		if contentType == "" {
			contentType = "application/json"
		}
		converted := &spec.Response{
			Description: resp.Description,
		}
		if schema != nil && resp.Type != nil {
			converted.Content = map[string]spec.MediaType{
				contentType: {Schema: schema},
			}
		}
		if len(resp.Headers) > 0 {
			converted.Headers = resp.Headers
		}
		out.Responses[resp.status] = converted
	}
	if len(op.SecurityReqs) > 0 {
		out.Security = op.SecurityReqs
	}
	if len(op.OptionalSecs) > 0 {
		// An optional security requirement means the endpoint accepts both
		// authenticated and anonymous calls, which OpenAPI expresses with
		// an empty requirement alongside the named ones.
		out.Security = append(out.Security, spec.SecurityRequirement{})
		out.Security = append(out.Security, op.OptionalSecs...)
	}

	return out
}

// sortedResponses returns the responses ordered by status code, so that the
// generated document is stable across runs.
func sortedResponses(responses map[string]*Response) []*Response {
	keys := make([]string, 0, len(responses))
	for k := range responses {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]*Response, 0, len(keys))
	for _, k := range keys {
		out = append(out, responses[k])
	}
	return out
}

// stripPrefix removes a leading path prefix from every registered path, so
// that a router mounted under a prefix produces paths relative to it.
func (r *Registry) stripPrefix(prefix string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		return
	}
	for _, op := range r.operations {
		if strings.HasPrefix(op.Path, prefix) {
			op.Path = strings.TrimPrefix(op.Path, prefix)
			if op.Path == "" {
				op.Path = "/"
			}
		}
	}
}
