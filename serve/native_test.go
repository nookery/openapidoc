package serve

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNativeUIRendersPage(t *testing.T) {
	ui := NativeUI()

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	ui.Serve(rec, req, UIContext{
		Title:      "Test API",
		SpecURL:    "http://localhost:8080/docs/openapi.json",
		AssetsPath: "/docs/_assets",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, "<html") {
		t.Error("response is not an HTML page")
	}
	if !strings.Contains(body, "/docs/_assets/app.css") {
		t.Error("page does not reference the stylesheet")
	}
	if !strings.Contains(body, "/docs/_assets/app.js") {
		t.Error("page does not reference the script")
	}
	if !strings.Contains(body, "http://localhost:8080/docs/openapi.json") {
		t.Error("page does not contain the spec URL")
	}
	if !strings.Contains(body, "Test API") {
		t.Error("page does not contain the title")
	}
}

func TestNativeUIHasNoExternalRequests(t *testing.T) {
	ui := NativeUI()

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	ui.Serve(rec, req, UIContext{
		Title:      "Test API",
		SpecURL:    "http://localhost:8080/docs/openapi.json",
		AssetsPath: "/docs/_assets",
	})

	body := rec.Body.String()

	// A self contained page must not reference any remote origin.
	for _, forbidden := range []string{
		"http://cdn",
		"https://cdn",
		"//cdn.",
		"unpkg.com",
		"jsdelivr.net",
		"googleapis.com",
		"http://localhost:3000",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("page references the external resource %q", forbidden)
		}
	}
}

func TestNativeAssetsHandlerServesFiles(t *testing.T) {
	handler := NativeAssetsHandler("/docs/_assets")

	files := []string{"app.css", "app.js", "openapi.js", "codegen.js"}
	for _, name := range files {
		req := httptest.NewRequest(http.MethodGet, "/docs/_assets/"+name, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", name, rec.Code)
			continue
		}
		if rec.Body.Len() == 0 {
			t.Errorf("%s: body is empty", name)
		}
	}
}

func TestNativeAssetsHandlerRejectsOtherPaths(t *testing.T) {
	handler := NativeAssetsHandler("/docs/_assets")

	cases := []string{
		"/other/app.css",
		"/docs/_assets/missing.css",
		"/docs/_assets/",
		"/",
	}
	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", path, rec.Code)
		}
	}
}

func TestAssetsPathOfCustomUI(t *testing.T) {
	// A plain function has no assets.
	if got := assetsPathOf(UIFunc(func(http.ResponseWriter, *http.Request, UIContext) {}), UIContext{}); got != "" {
		t.Errorf("assetsPathOf = %q, want empty", got)
	}

	// The bundled interface reports its asset prefix.
	if got := assetsPathOf(NativeUI(), UIContext{AssetsPath: "/docs/_assets"}); got != "/docs/_assets" {
		t.Errorf("assetsPathOf = %q, want /docs/_assets", got)
	}
}

func TestNativeUIFallsBackToTheDefaultAssetPath(t *testing.T) {
	ui := NativeUI()

	// Without an explicit path the interface uses a relative one.
	native, ok := ui.(*nativeUI)
	if !ok {
		t.Fatalf("NativeUI = %T, want the native interface", ui)
	}
	if got := native.AssetsPath(UIContext{}); got != "_assets" {
		t.Errorf("AssetsPath = %q, want _assets", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	rec := httptest.NewRecorder()
	ui.Serve(rec, req, UIContext{
		Title:   "Test API",
		SpecURL: "http://localhost:8080/docs/openapi.json",
	})

	if !strings.Contains(rec.Body.String(), "_assets/app.css") {
		t.Error("the page does not reference the default asset path")
	}
}
func TestNativeAssetsHandlerServesTheIndex(t *testing.T) {
	handler := NativeAssetsHandler("/docs/_assets")

	req := httptest.NewRequest(http.MethodGet, "/docs/_assets/index.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<html") {
		t.Error("the body is not an HTML page")
	}
}

func TestNativeAssetsHandlerWithoutPrefix(t *testing.T) {
	// An empty prefix serves the files from the root.
	handler := NativeAssetsHandler("")

	req := httptest.NewRequest(http.MethodGet, "/app.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestMountPrefix(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg, Options{DocsPath: "/docs"})

	cases := []struct {
		path string
		want string
	}{
		{"/docs", ""},
		{"/docs/", ""},
		{"/api/docs", "/api"},
		{"/api/docs/", "/api"},
		{"/other", ""},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		if got := h.mountPrefix(req); got != c.want {
			t.Errorf("mountPrefix(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestAssetsPathFollowsMountPrefix(t *testing.T) {
	reg := newTestRegistry()
	h := New(reg, Options{DocsPath: "/docs"})

	req := httptest.NewRequest(http.MethodGet, "/api/docs/", nil)
	if got := h.assetsPath(req); got != "/api/docs/_assets" {
		t.Errorf("assetsPath = %q, want /api/docs/_assets", got)
	}
}
