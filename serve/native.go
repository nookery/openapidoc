package serve

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
)

// nativeAssets holds the bundled documentation interface.
//
// The assets are plain HTML, CSS and JavaScript with no external
// dependencies, so the binary stays self contained and the documentation
// works on an isolated network.
//
//go:embed assets
var nativeAssets embed.FS

// nativeFS is the asset filesystem rooted at the assets directory.
var nativeFS = func() fs.FS {
	sub, err := fs.Sub(nativeAssets, "assets")
	if err != nil {
		panic(fmt.Sprintf("openapidoc: cannot open embedded assets: %v", err))
	}
	return sub
}()

// nativeTemplate is the documentation page.
var nativeTemplate = template.Must(template.New("index.html").
	ParseFS(nativeFS, "index.html"))

// nativeConfig configures the bundled UI.
type nativeConfig struct {
	// Title is the document title shown in the browser tab.
	Title string

	// SpecURL is the URL of the OpenAPI document, ready to be embedded in
	// a JavaScript string literal.
	SpecURL template.JS

	// AssetsPath is the URL prefix the static files are served from.
	AssetsPath string
}

// nativeUI is the bundled documentation interface.
type nativeUI struct{}

// Serve implements UI.
func (n *nativeUI) Serve(w http.ResponseWriter, r *http.Request, ctx UIContext) {
	data := nativeConfig{
		Title:      ctx.Title,
		SpecURL:    template.JS(jsString(ctx.SpecURL)),
		AssetsPath: ctx.AssetsPath,
	}
	if data.AssetsPath == "" {
		data.AssetsPath = "_assets"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := nativeTemplate.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "failed to render documentation", http.StatusInternalServerError)
	}
}

// AssetsPath implements assetProvider.
func (n *nativeUI) AssetsPath(ctx UIContext) string {
	if ctx.AssetsPath != "" {
		return ctx.AssetsPath
	}
	return "_assets"
}

// jsString quotes a value so it can be embedded in a JavaScript string
// literal. The values are produced by the handler, not by user input, but
// quoting keeps the output valid in every case.
func jsString(value string) string {
	quoted := strconv.Quote(value)
	// strconv escapes the forward slash on some platforms, which is
	// harmless but noisy in the generated page.
	return strings.ReplaceAll(quoted, "\\/", "/")
}

// NativeUI returns the bundled documentation interface.
//
// The interface is written in plain HTML, CSS and JavaScript. Everything is
// embedded in the binary, so the documentation renders without any network
// access and without a build step.
func NativeUI() UI {
	return &nativeUI{}
}

// NativeAssetsHandler serves the static files of the bundled interface.
//
// The handler serves the files under the given path prefix, for example
// "/docs/_assets/". It returns nil when the prefix does not match the
// request, so it can be mounted next to the documentation without
// interfering with other routes.
func NativeAssetsHandler(prefix string) http.Handler {
	prefix = strings.TrimSuffix(prefix, "/")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if prefix != "" && !strings.HasPrefix(r.URL.Path, prefix+"/") {
			http.NotFound(w, r)
			return
		}

		name := strings.TrimPrefix(r.URL.Path, prefix+"/")
		if name == "" {
			http.NotFound(w, r)
			return
		}

		file, err := nativeFS.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}

		http.ServeContent(w, r, name, info.ModTime(), file.(readSeeker))
	})
}

// readSeeker is the interface http.ServeContent needs.
type readSeeker interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
}
