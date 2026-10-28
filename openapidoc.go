// Package openapidoc provides browser readable API documentation for Go
// services.
//
// Describe the API with the api package, register the operations, and
// mount the documentation on any HTTP router. There is no code generation
// step: the OpenAPI document is built at run time from the Go types you
// pass in.
//
// A minimal service looks like this:
//
//	reg := openapidoc.New("My API", "1.0.0")
//	reg.Register(
//		openapidoc.Get("/users/{id}").
//			Summary("Get a user").
//			PathParam("id", 0).
//			Resp(200, new(User)),
//	)
//
//	r := http.NewServeMux()
//	r.Handle("/docs/", openapidoc.Handler(reg))
//	http.ListenAndServe(":8080", r)
//
// The documentation is then available at http://localhost:8080/docs.
package openapidoc

import (
	"net/http"

	"github.com/nookery/openapidoc/api"
	"github.com/nookery/openapidoc/serve"
	"github.com/nookery/openapidoc/spec"
)

// Registry collects the operations of an API.
//
// Create one with New and pass it to Handler to serve the documentation.
type Registry = api.Registry

// Operation describes a single HTTP operation.
type Operation = api.Operation

// Document is an OpenAPI 3.1 document.
type Document = spec.Document

// Options configures the documentation handler.
type Options = serve.Options

// UI renders an interactive API reference.
type UI = serve.UI

// New creates a registry for the given API title and version.
func New(title, version string) *Registry {
	return api.NewRegistry(title, version)
}

// NewHandler creates a documentation handler for the registry.
//
// The returned handler serves both the documentation UI and the OpenAPI
// document. Mount it under the path returned by DocsPath.
func NewHandler(registry *Registry, opts ...serve.Options) *serve.Handler {
	return serve.New(registry, opts...)
}

// Handler returns an http.Handler that serves the documentation UI at
// docsPath, the OpenAPI document at docsPath + "/openapi.json" and the
// static files of the bundled interface.
//
// It is a convenience wrapper around NewHandler for services that do not
// need to customise the options.
func Handler(registry *Registry, docsPath string) http.Handler {
	h := NewHandler(registry, serve.Options{DocsPath: docsPath})
	mux := http.NewServeMux()
	mux.Handle(h.DocsPath()+"/", h.DocsHandler())
	mux.Handle(h.SpecPath(), h.SpecHandler())
	if assets := h.AssetsHandler(); assets != nil {
		mux.Handle(h.AssetsPath()+"/", assets)
	}
	return mux
}

// Get returns a new GET operation.
func Get(path string) *Operation { return api.Get(path) }

// Post returns a new POST operation.
func Post(path string) *Operation { return api.Post(path) }

// Put returns a new PUT operation.
func Put(path string) *Operation { return api.Put(path) }

// Patch returns a new PATCH operation.
func Patch(path string) *Operation { return api.Patch(path) }

// Delete returns a new DELETE operation.
func Delete(path string) *Operation { return api.Delete(path) }

// Path returns a new operation for the given HTTP method and path.
func Path(method, path string) *Operation { return api.Path(method, path) }
