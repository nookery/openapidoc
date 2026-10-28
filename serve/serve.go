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
// The UI defaults to the bundled interface. Pass a different UI with
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

// AssetsPath returns the URL path serving the static files of the bundled
// interface.
//
// It is empty when a custom UI is in use, because a custom interface is
// responsible for its own assets.
func (h *Handler) AssetsPath() string {
	return assetsPathOf(h.ui, h.uiContext(nil))
}

// AssetsHandler serves the static files of the bundled interface.
//
// It returns nil when a custom UI is in use.
func (h *Handler) AssetsHandler() http.Handler {
	prefix := h.AssetsPath()
	if prefix == "" {
		return nil
	}
	return NativeAssetsHandler(prefix)
}

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
		h.ui.Serve(w, r, h.uiContext(r))
	})
}

// uiContext builds the context handed to the UI.
func (h *Handler) uiContext(r *http.Request) UIContext {
	title := h.opts.Title
	if title == "" {
		title = h.registry.Document().Info.Title
	}
	return UIContext{
		Title:      title,
		SpecURL:    h.specURL(r),
		AssetsPath: h.assetsPath(r),
	}
}

// mountPrefix returns the path the documentation is mounted under, without
// a trailing slash.
//
// It is empty when the documentation sits at the root of the server.
func (h *Handler) mountPrefix(r *http.Request) string {
	if r == nil {
		return ""
	}
	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == h.opts.DocsPath {
		return ""
	}
	if strings.HasSuffix(path, h.opts.DocsPath) {
		return strings.TrimSuffix(path, h.opts.DocsPath)
	}
	return ""
}

// assetsPath computes the URL prefix of the static files.
//
// The prefix is root relative so that it stays valid behind a proxy and
// needs no escaping in the rendered page.
func (h *Handler) assetsPath(r *http.Request) string {
	return h.mountPrefix(r) + h.opts.DocsPath + "/_assets"
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
	return scheme + "://" + r.Host + h.mountPrefix(r) + h.opts.SpecPath
}
