package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------- handler ----------

func TestSpecHandlerServesTheDocument(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg)

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
	rec := httptest.NewRecorder()
	h.SpecHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("content type = %q", ct)
	}

	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("the body is not valid JSON: %v", err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Errorf("openapi = %v, want 3.1.0", doc["openapi"])
	}
}

func TestSpecHandlerCacheAge(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg, Options{CacheAge: 3600})

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
	rec := httptest.NewRecorder()
	h.SpecHandler().ServeHTTP(rec, req)

	if got := rec.Header().Get("Cache-Control"); got != "max-age=3600" {
		t.Errorf("cache control = %q, want max-age=3600", got)
	}
}

func TestSpecHandlerWithoutCacheAge(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg)

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
	rec := httptest.NewRecorder()
	h.SpecHandler().ServeHTTP(rec, req)

	if got := rec.Header().Get("Cache-Control"); got != "" {
		t.Errorf("cache control = %q, want empty", got)
	}
}

func TestDocsHandlerServesTheInterface(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg)

	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	rec := httptest.NewRecorder()
	h.DocsHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("content type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "<html") {
		t.Error("the body is not an HTML page")
	}
}

func TestHandlerPaths(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg)

	if h.DocsPath() != "/docs" {
		t.Errorf("DocsPath = %q, want /docs", h.DocsPath())
	}
	if h.SpecPath() != "/docs/openapi.json" {
		t.Errorf("SpecPath = %q, want /docs/openapi.json", h.SpecPath())
	}
	if h.AssetsPath() != "/docs/_assets" {
		t.Errorf("AssetsPath = %q, want /docs/_assets", h.AssetsPath())
	}
}

func TestHandlerCustomPaths(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg, Options{
		DocsPath: "/api-docs",
		SpecPath: "/api-docs/spec.json",
	})

	if h.DocsPath() != "/api-docs" {
		t.Errorf("DocsPath = %q", h.DocsPath())
	}
	if h.SpecPath() != "/api-docs/spec.json" {
		t.Errorf("SpecPath = %q", h.SpecPath())
	}
}

func TestHandlerAssetsHandler(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg)

	assets := h.AssetsHandler()
	if assets == nil {
		t.Fatal("AssetsHandler is nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/docs/_assets/app.css", nil)
	rec := httptest.NewRecorder()
	assets.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestHandlerAssetsHandlerIsNilForACustomUI(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg)
	h.WithUI(UIFunc(func(w http.ResponseWriter, r *http.Request, ctx UIContext) {
		w.WriteHeader(http.StatusOK)
	}))

	if h.AssetsPath() != "" {
		t.Errorf("AssetsPath = %q, want empty", h.AssetsPath())
	}
	if h.AssetsHandler() != nil {
		t.Error("AssetsHandler must be nil for a custom interface")
	}
}

func TestWithUIReplacesTheInterface(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg)

	h.WithUI(UIFunc(func(w http.ResponseWriter, r *http.Request, ctx UIContext) {
		w.Write([]byte("custom interface"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	rec := httptest.NewRecorder()
	h.DocsHandler().ServeHTTP(rec, req)

	if rec.Body.String() != "custom interface" {
		t.Errorf("body = %q, want the custom interface", rec.Body.String())
	}
}

// ---------- spec url ----------

func TestSpecURL(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg)

	cases := []struct {
		path      string
		host      string
		tls       bool
		forwarded string
		wantURL   string
	}{
		{
			path:    "/docs/",
			host:    "example.com",
			wantURL: "http://example.com/docs/openapi.json",
		},
		{
			path:    "/docs/",
			host:    "example.com",
			tls:     true,
			wantURL: "https://example.com/docs/openapi.json",
		},
		{
			path:      "/docs/",
			host:      "example.com",
			forwarded: "https",
			wantURL:   "https://example.com/docs/openapi.json",
		},
		{
			path:      "/docs/",
			host:      "example.com",
			forwarded: "https,http",
			wantURL:   "https://example.com/docs/openapi.json",
		},
		{
			path:    "/api/docs/",
			host:    "example.com",
			wantURL: "http://example.com/api/docs/openapi.json",
		},
	}

	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		req.Host = c.host
		if c.tls {
			req.TLS = &tlsState
		}
		if c.forwarded != "" {
			req.Header.Set("X-Forwarded-Proto", c.forwarded)
		}

		if got := h.specURL(req); got != c.wantURL {
			t.Errorf("specURL(%q) = %q, want %q", c.path, got, c.wantURL)
		}
	}
}

func TestSpecURLWithoutRequest(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg)

	if got := h.specURL(nil); got != "/docs/openapi.json" {
		t.Errorf("specURL = %q, want /docs/openapi.json", got)
	}
}

func TestUIContextUsesTheDocumentTitle(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg)

	ctx := h.uiContext(nil)
	if ctx.Title != "Test API" {
		t.Errorf("title = %q, want Test API", ctx.Title)
	}
	if ctx.AssetsPath != "/docs/_assets" {
		t.Errorf("assets path = %q, want /docs/_assets", ctx.AssetsPath)
	}
}

func TestUIContextUsesTheTitleOverride(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg, Options{Title: "Overridden"})

	if got := h.uiContext(nil).Title; got != "Overridden" {
		t.Errorf("title = %q, want Overridden", got)
	}
}

// ---------- ui ----------

func TestUIFuncServes(t *testing.T) {
	ui := UIFunc(func(w http.ResponseWriter, r *http.Request, ctx UIContext) {
		w.Write([]byte("hello"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ui.Serve(rec, req, UIContext{})

	if rec.Body.String() != "hello" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestDefaultUIIsTheNativeInterface(t *testing.T) {
	if _, ok := DefaultUI().(*nativeUI); !ok {
		t.Errorf("DefaultUI = %T, want the native interface", DefaultUI())
	}
}

// ---------- third party interfaces ----------

func TestScalarUIRenders(t *testing.T) {
	ui := ScalarUI()

	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	rec := httptest.NewRecorder()
	ui.Serve(rec, req, UIContext{
		Title:   "Test API",
		SpecURL: "http://example.com/docs/openapi.json",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<html") {
		t.Error("the body is not an HTML page")
	}
	if !strings.Contains(body, "http://example.com/docs/openapi.json") {
		t.Error("the page does not reference the spec URL")
	}
	if !strings.Contains(body, "Test API") {
		t.Error("the page does not contain the title")
	}
}

func TestScalarUIWithCustomConfig(t *testing.T) {
	ui := ScalarUI(ScalarConfig{
		Theme:      "purple",
		Layout:     "classic",
		DarkMode:   true,
		CDNBaseURL: "https://mirror.example.com",
	})

	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	rec := httptest.NewRecorder()
	ui.Serve(rec, req, UIContext{Title: "T", SpecURL: "http://example.com/spec.json"})

	body := rec.Body.String()
	if !strings.Contains(body, "https://mirror.example.com") {
		t.Error("the custom CDN URL is not used")
	}
}

func TestRedocUIRenders(t *testing.T) {
	ui := RedocUI()

	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	rec := httptest.NewRecorder()
	ui.Serve(rec, req, UIContext{
		Title:   "Test API",
		SpecURL: "http://example.com/docs/openapi.json",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<redoc") {
		t.Error("the page does not contain the redoc element")
	}
	if !strings.Contains(body, "http://example.com/docs/openapi.json") {
		t.Error("the page does not reference the spec URL")
	}
}

func TestRedocUIWithCustomConfig(t *testing.T) {
	ui := RedocUI(RedocConfig{CDNBaseURL: "https://mirror.example.com"})

	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	rec := httptest.NewRecorder()
	ui.Serve(rec, req, UIContext{Title: "T", SpecURL: "http://example.com/spec.json"})

	if !strings.Contains(rec.Body.String(), "https://mirror.example.com") {
		t.Error("the custom CDN URL is not used")
	}
}

func TestThirdPartyUIEscapesTheTitle(t *testing.T) {
	ui := RedocUI()

	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	rec := httptest.NewRecorder()
	ui.Serve(rec, req, UIContext{
		Title:   `<script>alert("x")</script>`,
		SpecURL: "http://example.com/spec.json",
	})

	if strings.Contains(rec.Body.String(), "<script>alert") {
		t.Error("the title must be escaped")
	}
}
