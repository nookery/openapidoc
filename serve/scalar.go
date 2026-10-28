package serve

import (
	"fmt"
	"html/template"
	"net/http"
)

// ScalarConfig configures the Scalar documentation UI.
type ScalarConfig struct {
	// Theme is the Scalar theme name, for example "default", "purple",
	// "moon" or "mars". Defaults to "default".
	Theme string

	// Layout is either "modern" or "classic". Defaults to "modern".
	Layout string

	// DarkMode enables the dark colour scheme by default.
	DarkMode bool

	// HideSearch hides the search bar.
	HideSearch bool

	// HideModels hides the models section in the sidebar.
	HideModels bool

	// HideTestRequestButton hides the try it panel.
	HideTestRequestButton bool

	// CDNBaseURL is the base URL of the Scalar assets. Change it to serve
	// the assets from a mirror or a local path.
	CDNBaseURL string
}

// defaultScalarCDN is the CDN that serves the Scalar browser bundle.
const defaultScalarCDN = "https://cdn.jsdelivr.net/npm/@scalar/api-reference"

// scalarTemplate renders the documentation page.
//
// The Scalar bundle is loaded from a CDN by default. To make the binary
// self contained and usable on an isolated network, use the embedded
// variant of this package.
var scalarTemplate = template.Must(template.New("scalar").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<link rel="stylesheet" href="{{.CDNBaseURL}}/style.css">
</head>
<body>
<script id="api-reference" data-url="{{.SpecURL}}"></script>
<script src="{{.CDNBaseURL}}"></script>
</body>
</html>
`))

// ScalarUI returns a documentation UI backed by Scalar.
//
// The assets are loaded from a CDN, so the documentation needs network
// access the first time it is opened. Use the embedded variant for offline
// environments.
func ScalarUI(config ...ScalarConfig) UI {
	cfg := ScalarConfig{}
	if len(config) > 0 {
		cfg = config[0]
	}
	if cfg.CDNBaseURL == "" {
		cfg.CDNBaseURL = defaultScalarCDN
	}
	if cfg.Theme == "" {
		cfg.Theme = "default"
	}
	if cfg.Layout == "" {
		cfg.Layout = "modern"
	}

	return UIFunc(func(w http.ResponseWriter, r *http.Request, ctx UIContext) {
		data := map[string]any{
			"Title":      ctx.Title,
			"SpecURL":    ctx.SpecURL,
			"CDNBaseURL": cfg.CDNBaseURL,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := scalarTemplate.Execute(w, data); err != nil {
			http.Error(w, "failed to render documentation", http.StatusInternalServerError)
		}
	})
}

// RedocUI returns a documentation UI backed by ReDoc.
//
// ReDoc renders a three column reference that is optimised for reading
// rather than for trying out requests.
func RedocUI(config ...RedocConfig) UI {
	cfg := RedocConfig{}
	if len(config) > 0 {
		cfg = config[0]
	}
	if cfg.CDNBaseURL == "" {
		cfg.CDNBaseURL = "https://cdn.jsdelivr.net/npm/redoc@latest/bundles"
	}

	return UIFunc(func(w http.ResponseWriter, r *http.Request, ctx UIContext) {
		title := ctx.Title
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>body{margin:0;padding:0}</style>
</head>
<body>
<redoc spec-url="%s"></redoc>
<script src="%s/redoc.standalone.js"></script>
</body>
</html>
`, template.HTMLEscapeString(title), template.HTMLEscapeString(ctx.SpecURL), cfg.CDNBaseURL)
	})
}

// RedocConfig configures the ReDoc documentation UI.
type RedocConfig struct {
	// CDNBaseURL is the base URL of the ReDoc bundle.
	CDNBaseURL string
}
