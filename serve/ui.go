package serve

import (
	"net/http"
)

// UIContext carries the information a documentation UI needs to render.
type UIContext struct {
	// Title is the document title shown in the browser tab and header.
	Title string

	// SpecURL is the absolute URL of the OpenAPI document.
	SpecURL string
}

// UI renders an interactive API reference.
//
// Implement this interface to serve a custom documentation interface.
type UI interface {
	// Serve writes the documentation UI to the response.
	Serve(w http.ResponseWriter, r *http.Request, ctx UIContext)
}

// UIFunc adapts a function to the UI interface.
type UIFunc func(w http.ResponseWriter, r *http.Request, ctx UIContext)

// Serve implements UI.
func (f UIFunc) Serve(w http.ResponseWriter, r *http.Request, ctx UIContext) {
	f(w, r, ctx)
}

// DefaultUI returns the documentation UI used when none is configured.
//
// The default is Scalar, which renders a complete interactive reference with
// a built in API client, code samples and search.
func DefaultUI() UI {
	return ScalarUI()
}
