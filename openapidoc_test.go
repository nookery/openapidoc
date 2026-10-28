package openapidoc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nookery/openapidoc"
)

// User is a sample resource used by the tests.
type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	Age       *int   `json:"age,omitempty"`
	CreatedAt string `json:"created_at"`
}

// CreateUserRequest is the payload of the create endpoint.
type CreateUserRequest struct {
	Name  string `json:"name" jsonschema:"description=Display name,required"`
	Email string `json:"email" jsonschema:"description=Contact address"`
	Age   int    `json:"age,omitempty" jsonschema:"description=Age in years,minimum=0,maximum=150"`
}

// ErrorResponse is the standard error payload.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// newRegistry builds a registry with a few representative operations.
func newRegistry() *openapidoc.Registry {
	reg := openapidoc.New("Test API", "1.0.0")
	reg.Description("A test API")
	reg.AddServer("http://localhost:8080", "Local development")
	reg.AddTag("users", "User management")
	reg.BearerAuth("bearerAuth", "JWT authentication")

	reg.Register(
		openapidoc.Get("/users/{id}").
			Summary("Get a user").
			Describe("Returns a single user by its identifier.").
			Tag("users").
			PathParam("id", 0).
			Security("bearerAuth").
			Resp(200, new(User)).
			Resp(404, new(ErrorResponse)),

		openapidoc.Post("/users").
			Summary("Create a user").
			Tag("users").
			Body(new(CreateUserRequest)).
			Resp(201, new(User)).
			Resp(400, new(ErrorResponse)),

		openapidoc.Get("/users").
			Summary("List users").
			Tag("users").
			Query("page", 0).
			Query("page_size", 20).
			Security("bearerAuth").
			Resp(200, new([]User)),
	)

	return reg
}

func TestDocumentStructure(t *testing.T) {
	doc := newRegistry().Document()

	if doc.OpenAPI != "3.1.0" {
		t.Errorf("OpenAPI version = %q, want 3.1.0", doc.OpenAPI)
	}
	if doc.Info.Title != "Test API" {
		t.Errorf("title = %q, want Test API", doc.Info.Title)
	}
	if len(doc.Paths) != 2 {
		t.Fatalf("paths = %d, want 2", len(doc.Paths))
	}
	if _, ok := doc.Paths["/users/{id}"]; !ok {
		t.Error("missing path /users/{id}")
	}
	if _, ok := doc.Paths["/users"]; !ok {
		t.Error("missing path /users")
	}
}

func TestOperationDetails(t *testing.T) {
	doc := newRegistry().Document()

	item := doc.Paths["/users/{id}"]
	if item.Get == nil {
		t.Fatal("GET /users/{id} is missing")
	}
	op := item.Get

	if op.Summary != "Get a user" {
		t.Errorf("summary = %q, want Get a user", op.Summary)
	}
	if len(op.Tags) != 1 || op.Tags[0] != "users" {
		t.Errorf("tags = %v, want [users]", op.Tags)
	}
	if len(op.Parameters) != 1 {
		t.Fatalf("parameters = %d, want 1", len(op.Parameters))
	}
	p := op.Parameters[0]
	if p.Name != "id" || p.In != "path" {
		t.Errorf("parameter = %+v, want id in path", p)
	}
	if !p.Required {
		t.Error("path parameter must be required")
	}
	if len(op.Responses) != 2 {
		t.Errorf("responses = %d, want 2", len(op.Responses))
	}
	if _, ok := op.Responses["200"]; !ok {
		t.Errorf("missing 200 response, got %v", keys(op.Responses))
	}
	if len(op.Security) != 1 {
		t.Errorf("security = %v, want one requirement", op.Security)
	}
}

func TestComponentSchemas(t *testing.T) {
	doc := newRegistry().Document()

	if doc.Components == nil {
		t.Fatal("components is nil")
	}
	schemas := doc.Components.Schemas
	if len(schemas) == 0 {
		t.Fatal("no component schemas")
	}

	user, ok := schemas["User"]
	if !ok {
		t.Fatalf("User schema is missing, got %v", keys(schemas))
	}
	if user.Properties["id"] == nil {
		t.Error("User.id property is missing")
	}
	if user.Properties["id"].Type != "integer" {
		t.Errorf("User.id type = %q, want integer", user.Properties["id"].Type)
	}
	if user.Properties["email"].Type != "string" {
		t.Errorf("User.email type = %q, want string", user.Properties["email"].Type)
	}

	// A pointer field is optional, a plain field is required.
	var required []string
	for _, name := range user.Required {
		required = append(required, name)
	}
	if !contains(required, "id") {
		t.Errorf("User required = %v, want it to contain id", required)
	}
	if contains(required, "age") {
		t.Errorf("User required = %v, want it to exclude the optional age", required)
	}
}

func TestStructTagOverrides(t *testing.T) {
	doc := newRegistry().Document()

	req, ok := doc.Components.Schemas["CreateUserRequest"]
	if !ok {
		t.Fatal("CreateUserRequest schema is missing")
	}
	if got := req.Properties["name"].Description; got != "Display name" {
		t.Errorf("name description = %q, want Display name", got)
	}
	if got := req.Properties["age"].Minimum; got == nil || *got != 0 {
		t.Errorf("age minimum = %v, want 0", got)
	}
	if got := req.Properties["age"].Maximum; got == nil || *got != 150 {
		t.Errorf("age maximum = %v, want 150", got)
	}
}

func TestRecursiveTypeTerminates(t *testing.T) {
	reg := openapidoc.New("Recursive API", "1.0.0")
	reg.Register(
		openapidoc.Get("/tree").
			Summary("Get a tree").
			Resp(200, new(Node)),
	)

	doc := reg.Document()
	node, ok := doc.Components.Schemas["Node"]
	if !ok {
		t.Fatal("Node schema is missing")
	}
	child := node.Properties["child"]
	if child == nil {
		t.Fatal("Node.child is missing")
	}
	if child.Ref == "" {
		t.Errorf("Node.child should be a reference, got %+v", child)
	}
	if child.Ref != "#/components/schemas/Node" {
		t.Errorf("Node.child ref = %q, want #/components/schemas/Node", child.Ref)
	}
}

// Node is a self referencing type.
type Node struct {
	Name  string `json:"name"`
	Child *Node  `json:"child,omitempty"`
}

func TestSpecHandlerServesJSON(t *testing.T) {
	reg := newRegistry()
	handler := openapidoc.NewHandler(reg)

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
	rec := httptest.NewRecorder()
	handler.SpecHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" {
		t.Error("Content-Type header is missing")
	}

	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Errorf("openapi = %v, want 3.1.0", doc["openapi"])
	}
}

func TestDocsHandlerRendersUI(t *testing.T) {
	reg := newRegistry()
	handler := openapidoc.NewHandler(reg)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	handler.DocsHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if body == "" {
		t.Fatal("body is empty")
	}
	if !containsSubstring(body, "<html") {
		t.Error("response does not look like an HTML page")
	}
	if !containsSubstring(body, "/docs/openapi.json") {
		t.Error("page does not reference the spec URL")
	}
}

func TestHandlerPaths(t *testing.T) {
	reg := newRegistry()
	h := openapidoc.NewHandler(reg, openapidoc.Options{DocsPath: "/api-docs"})

	if h.DocsPath() != "/api-docs" {
		t.Errorf("DocsPath = %q, want /api-docs", h.DocsPath())
	}
	if h.SpecPath() != "/api-docs/openapi.json" {
		t.Errorf("SpecPath = %q, want /api-docs/openapi.json", h.SpecPath())
	}
}

func TestHandlerServesAssets(t *testing.T) {
	reg := newRegistry()
	h := openapidoc.NewHandler(reg)

	if h.AssetsPath() != "/docs/_assets" {
		t.Errorf("AssetsPath = %q, want /docs/_assets", h.AssetsPath())
	}

	assets := h.AssetsHandler()
	if assets == nil {
		t.Fatal("AssetsHandler is nil")
	}

	for _, name := range []string{"app.css", "app.js", "openapi.js", "codegen.js"} {
		req := httptest.NewRequest(http.MethodGet, "/docs/_assets/"+name, nil)
		rec := httptest.NewRecorder()
		assets.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", name, rec.Code)
		}
	}
}

func TestHandlerConvenienceFunction(t *testing.T) {
	reg := newRegistry()
	handler := openapidoc.Handler(reg, "/docs")

	// The documentation page renders.
	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("docs: status = %d, want 200", rec.Code)
	}

	// The spec is served.
	req = httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("spec: status = %d, want 200", rec.Code)
	}

	// The static files are served.
	req = httptest.NewRequest(http.MethodGet, "/docs/_assets/app.css", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("assets: status = %d, want 200", rec.Code)
	}
}

// contains reports whether the slice holds the value.
func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

// containsSubstring reports whether s contains sub.
func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

// indexOf returns the index of sub in s, or -1.
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// keys returns the map keys as a slice.
func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
