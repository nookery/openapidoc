package serve

import (
	"net/http"
)

// UIContext carries the information a documentation UI needs to render.
type UIContext struct {
	// Title is the document title shown in the browser tab and header.
	Title string

	// SpecURL is the URL of the OpenAPI document.
	SpecURL string

	// AssetsPath is the URL prefix the static files of the UI are served
	// from. It is empty when the UI ships no static files.
	AssetsPath string
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

// assetProvider is implemented by UIs that ship their own static files.
type assetProvider interface {
	// AssetsPath returns the URL prefix the files are served from.
	AssetsPath(ctx UIContext) string
}

// assetsPathOf returns the asset prefix of a UI, or an empty string when
// the UI does not serve any static files.
func assetsPathOf(ui UI, ctx UIContext) string {
	if provider, ok := ui.(assetProvider); ok {
		return provider.AssetsPath(ctx)
	}
	return ""
}

// DefaultUI returns the documentation UI used when none is configured.
//
// The default is the bundled interface, which is written in plain HTML, CSS
// and JavaScript and embedded in the binary. It needs no network access and
// no build step.
func DefaultUI() UI {
	return NativeUI()
}
