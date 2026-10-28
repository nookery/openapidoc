// Package serve exposes an OpenAPI document and a documentation UI over
// HTTP.
//
// The handlers use only the standard library, so they can be mounted on any
// router: net/http, gin, echo, chi and others.
package serve

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/nookery/openapidoc/api"
)

// Options configures a Handler.
type Options struct {
	// DocsPath is the URL path serving the documentation UI.
	// Defaults to "/docs".
	DocsPath string

	// SpecPath is the URL path serving the OpenAPI document.
	// Defaults to "/docs/openapi.json".
	SpecPath string

	// Title overrides the document title shown in the UI.
	Title string

	// CacheAge is the Cache-Control max-age, in seconds, applied to the
	// spec response. Zero disables caching headers.
	CacheAge int
}

// Handler serves the OpenAPI document and the documentation UI.
type Handler struct {
	registry *api.Registry
	opts     Options
	ui       UI
}

// New creates a Handler for the given registry.
//
// The UI defaults to the bundled Scalar reference. Pass a different UI with
// WithUI to change the rendering.
func New(registry *api.Registry, opts ...Options) *Handler {
	h := &Handler{
		registry: registry,
		opts:     Options{},
		ui:       DefaultUI(),
	}
	for _, o := range opts {
		h.opts.DocsPath = o.DocsPath
		h.opts.SpecPath = o.SpecPath
		h.opts.Title = o.Title
		h.opts.CacheAge = o.CacheAge
	}
	if h.opts.DocsPath == "" {
		h.opts.DocsPath = "/docs"
	}
	if h.opts.SpecPath == "" {
		h.opts.SpecPath = h.opts.DocsPath + "/openapi.json"
	}
	return h
}

// WithUI replaces the documentation UI.
func (h *Handler) WithUI(ui UI) *Handler {
	h.ui = ui
	return h
}

// DocsPath returns the URL path serving the documentation UI.
func (h *Handler) DocsPath() string { return h.opts.DocsPath }

// SpecPath returns the URL path serving the OpenAPI document.
func (h *Handler) SpecPath() string { return h.opts.SpecPath }

// SpecHandler serves the OpenAPI document as JSON.
func (h *Handler) SpecHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		doc := h.registry.Document()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if h.opts.CacheAge > 0 {
			w.Header().Set("Cache-Control", "max-age="+strconv.Itoa(h.opts.CacheAge))
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(doc); err != nil {
			http.Error(w, "failed to encode OpenAPI document", http.StatusInternalServerError)
		}
	})
}

// DocsHandler serves the documentation UI.
func (h *Handler) DocsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		title := h.opts.Title
		if title == "" {
			title = h.registry.Document().Info.Title
		}
		h.ui.Serve(w, r, UIContext{
			Title:   title,
			SpecURL: h.specURL(r),
		})
	})
}

// specURL computes the absolute URL of the spec endpoint as seen by the
// browser, so that the UI can fetch it regardless of any path prefix.
func (h *Handler) specURL(r *http.Request) string {
	if r == nil {
		return h.opts.SpecPath
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Prefer the forwarded scheme when the app runs behind a proxy.
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = strings.Split(fwd, ",")[0]
	}

	prefix := strings.TrimSuffix(r.URL.Path, h.opts.DocsPath)
	if prefix == "" {
		prefix = ""
	}
	return scheme + "://" + r.Host + prefix + h.opts.SpecPath
}
