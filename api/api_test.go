package api

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/nookery/openapidoc/spec"
)

// ---------- fixtures ----------

type Payload struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type OtherPayload struct {
	Ref string `json:"ref"`
}

// ---------- operation builders ----------

func TestPathNormalisesMethod(t *testing.T) {
	cases := []struct {
		method string
		want   string
	}{
		{"get", http.MethodGet},
		{"POST", http.MethodPost},
		{"put", http.MethodPut},
		{"delete", http.MethodDelete},
		{"", http.MethodGet},
	}
	for _, c := range cases {
		op := Path(c.method, "/x")
		if op.Method != c.want {
			t.Errorf("Path(%q).Method = %q, want %q", c.method, op.Method, c.want)
		}
	}
}

func TestMethodHelpers(t *testing.T) {
	cases := []struct {
		op     *Operation
		method string
	}{
		{Get("/x"), http.MethodGet},
		{Post("/x"), http.MethodPost},
		{Put("/x"), http.MethodPut},
		{Patch("/x"), http.MethodPatch},
		{Delete("/x"), http.MethodDelete},
	}
	for _, c := range cases {
		if c.op.Method != c.method {
			t.Errorf("method = %q, want %q", c.op.Method, c.method)
		}
		if c.op.Path != "/x" {
			t.Errorf("path = %q, want /x", c.op.Path)
		}
	}
}

func TestOperationMetadata(t *testing.T) {
	op := Get("/x").
		Summary("A summary").
		Describe("A description").
		ID("opId").
		Tag("a", "b").
		MarkDeprecated()

	if op.SummaryText != "A summary" {
		t.Errorf("summary = %q", op.SummaryText)
	}
	if op.Description != "A description" {
		t.Errorf("description = %q", op.Description)
	}
	if op.OperationID != "opId" {
		t.Errorf("operationId = %q", op.OperationID)
	}
	if len(op.Tags) != 2 || op.Tags[0] != "a" || op.Tags[1] != "b" {
		t.Errorf("tags = %v, want [a b]", op.Tags)
	}
	if !op.Deprecated {
		t.Error("deprecated is not set")
	}
}

func TestOperationParameterLocations(t *testing.T) {
	op := Get("/x").
		Query("q", "").
		RequiredQuery("rq", 0).
		Header("X-Token", "").
		PathParam("id", 0).
		Cookie("session", "")

	if len(op.Parameters) != 5 {
		t.Fatalf("parameters = %d, want 5", len(op.Parameters))
	}

	want := []struct {
		name     string
		in       string
		required bool
	}{
		{"q", "query", false},
		{"rq", "query", true},
		{"X-Token", "header", false},
		{"id", "path", true},
		{"session", "cookie", false},
	}
	for i, w := range want {
		p := op.Parameters[i]
		if p.Name != w.name {
			t.Errorf("parameter %d name = %q, want %q", i, p.Name, w.name)
		}
		if p.In != w.in {
			t.Errorf("parameter %d in = %q, want %q", i, p.In, w.in)
		}
		if p.Required != w.required {
			t.Errorf("parameter %d required = %v, want %v", i, p.Required, w.required)
		}
	}
}

func TestParameterTypeIsCaptured(t *testing.T) {
	op := Get("/x").Query("page", 1).PathParam("id", 0)

	if op.Parameters[0].Type != reflect.TypeOf(int(0)) {
		t.Errorf("query type = %v, want int", op.Parameters[0].Type)
	}
	if op.Parameters[1].Type != reflect.TypeOf(int(0)) {
		t.Errorf("path type = %v, want int", op.Parameters[1].Type)
	}
}

func TestParamWithExplicitSchema(t *testing.T) {
	schema := &spec.Schema{Type: "string", Format: "uuid"}
	op := Get("/x").Param(Parameter{
		Name:        "id",
		In:          "query",
		Description: "Identifier",
		Required:    true,
		Schema:      schema,
		Example:     "abc",
	})

	if len(op.Parameters) != 1 {
		t.Fatalf("parameters = %d, want 1", len(op.Parameters))
	}
	p := op.Parameters[0]
	if p.Schema != schema {
		t.Error("the explicit schema was not kept")
	}
	if p.Example != "abc" {
		t.Errorf("example = %v, want abc", p.Example)
	}
}

func TestBodyVariants(t *testing.T) {
	op := Post("/x").
		Body(new(Payload)).
		OptionalBody(new(OtherPayload)).
		BodyWith("application/xml", new(Payload))

	if op.RequestBody == nil {
		t.Fatal("request body is nil")
	}
	if op.RequestBody.ContentType != "application/xml" {
		t.Errorf("content type = %q, want application/xml", op.RequestBody.ContentType)
	}
	if !op.RequestBody.Required {
		t.Error("the body must be required")
	}
	if op.RequestBody.Type != reflect.TypeOf(Payload{}) {
		t.Errorf("body type = %v, want Payload", op.RequestBody.Type)
	}
}

func TestOptionalBodyIsNotRequired(t *testing.T) {
	op := Post("/x").OptionalBody(new(Payload))
	if op.RequestBody.Required {
		t.Error("an optional body must not be required")
	}
}

func TestRespUsesNumericStatusKey(t *testing.T) {
	op := Get("/x").Resp(200, new(Payload)).Resp(404, new(OtherPayload))

	if _, ok := op.Responses["200"]; !ok {
		t.Errorf("responses = %v, want a 200 key", keys(op.Responses))
	}
	if _, ok := op.Responses["404"]; !ok {
		t.Errorf("responses = %v, want a 404 key", keys(op.Responses))
	}
}

func TestRespWithCustomContentType(t *testing.T) {
	op := Get("/x").RespWith(200, "text/csv", nil)
	resp := op.Responses["200"]
	if resp == nil {
		t.Fatal("the response is missing")
	}
	if resp.ContentType != "text/csv" {
		t.Errorf("content type = %q, want text/csv", resp.ContentType)
	}
	if resp.Type != nil {
		t.Errorf("type = %v, want nil for a nil value", resp.Type)
	}
}

func TestRespHeaderAppliesToEveryResponse(t *testing.T) {
	op := Get("/x").
		Resp(200, new(Payload)).
		Resp(404, new(OtherPayload)).
		RespHeader("X-Rate-Limit", "Remaining quota", "")

	for code, resp := range op.Responses {
		header, ok := resp.Headers["X-Rate-Limit"]
		if !ok {
			t.Errorf("response %s has no header", code)
			continue
		}
		if header.Description != "Remaining quota" {
			t.Errorf("description = %q", header.Description)
		}
		if header.Schema == nil || header.Schema.Type != "string" {
			t.Errorf("schema = %+v, want string", header.Schema)
		}
	}
}

func TestSecurityRequirements(t *testing.T) {
	op := Get("/x").Security("bearer", "apiKey")

	if len(op.SecurityReqs) != 1 {
		t.Fatalf("security = %d, want 1", len(op.SecurityReqs))
	}
	req := op.SecurityReqs[0]
	if len(req) != 2 {
		t.Errorf("schemes = %v, want two", req)
	}
	if _, ok := req["bearer"]; !ok {
		t.Error("the bearer scheme is missing")
	}
	if _, ok := req["apiKey"]; !ok {
		t.Error("the apiKey scheme is missing")
	}
}

func TestOptionalSecurity(t *testing.T) {
	op := Get("/x").SecurityOptional("bearer")
	if len(op.OptionalSecs) != 1 {
		t.Fatalf("optional security = %d, want 1", len(op.OptionalSecs))
	}
}

// ---------- registry ----------

func TestRegistryMetadata(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0").
		Title("New Title").
		Version("2.0.0").
		Description("A description").
		AddServer("https://api.example.com", "Production").
		AddServer("http://localhost", "Local").
		AddTag("users", "User management")

	doc := reg.Document()

	if doc.OpenAPI != "3.1.0" {
		t.Errorf("openapi = %q, want 3.1.0", doc.OpenAPI)
	}
	if doc.Info.Title != "New Title" {
		t.Errorf("title = %q", doc.Info.Title)
	}
	if doc.Info.Version != "2.0.0" {
		t.Errorf("version = %q", doc.Info.Version)
	}
	if doc.Info.Description != "A description" {
		t.Errorf("description = %q", doc.Info.Description)
	}
	if len(doc.Servers) != 2 {
		t.Errorf("servers = %d, want 2", len(doc.Servers))
	}
	if len(doc.Tags) != 1 || doc.Tags[0].Name != "users" {
		t.Errorf("tags = %v", doc.Tags)
	}
}

func TestRegistryRegistersOperations(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(Get("/a"), Post("/b"))
	reg.Register(nil)

	if len(reg.Operations()) != 2 {
		t.Errorf("operations = %d, want 2", len(reg.Operations()))
	}
}

func TestOperationsReturnsACopy(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(Get("/a"))

	ops := reg.Operations()
	ops[0] = nil

	if reg.Operations()[0] == nil {
		t.Error("Operations must return a copy")
	}
}

func TestDocumentGroupsOperationsByPath(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(
		Get("/users/{id}"),
		Post("/users/{id}"),
		Delete("/users/{id}"),
		Get("/posts"),
	)

	doc := reg.Document()
	if len(doc.Paths) != 2 {
		t.Fatalf("paths = %d, want 2", len(doc.Paths))
	}

	item := doc.Paths["/users/{id}"]
	if item.Get == nil || item.Post == nil || item.Delete == nil {
		t.Errorf("the path item is incomplete: %+v", item)
	}
	if item.Put != nil || item.Patch != nil {
		t.Error("an undeclared method must be nil")
	}
}

func TestDocumentComponentSchemas(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(Get("/users").Resp(200, new(Payload)))

	doc := reg.Document()
	if doc.Components == nil {
		t.Fatal("components are missing")
	}
	if _, ok := doc.Components.Schemas["Payload"]; !ok {
		t.Errorf("schemas = %v, want a Payload component", keys(doc.Components.Schemas))
	}
}

func TestDocumentSecuritySchemes(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.BearerAuth("bearerAuth", "JWT token")
	reg.APIKeyAuth("apiKey", "X-API-Key", "API key")
	reg.Register(Get("/x").Security("bearerAuth"))

	doc := reg.Document()
	if doc.Components == nil {
		t.Fatal("components are missing")
	}
	schemes := doc.Components.SecuritySchemes
	if len(schemes) != 2 {
		t.Fatalf("schemes = %d, want 2", len(schemes))
	}

	bearer := schemes["bearerAuth"]
	if bearer == nil {
		t.Fatal("the bearerAuth scheme is missing")
	}
	if bearer.Type != "http" || bearer.Scheme != "bearer" {
		t.Errorf("bearer = %+v", bearer)
	}
	if bearer.BearerFormat != "JWT" {
		t.Errorf("bearerFormat = %q, want JWT", bearer.BearerFormat)
	}

	apiKey := schemes["apiKey"]
	if apiKey == nil {
		t.Fatal("the apiKey scheme is missing")
	}
	if apiKey.Type != "apiKey" || apiKey.Name != "X-API-Key" || apiKey.In != "header" {
		t.Errorf("apiKey = %+v", apiKey)
	}
}

func TestDocumentOptionalSecurityIncludesAnEmptyRequirement(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.BearerAuth("bearerAuth", "JWT token")
	reg.Register(Get("/x").SecurityOptional("bearerAuth"))

	doc := reg.Document()
	op := doc.Paths["/x"].Get
	if op == nil {
		t.Fatal("the operation is missing")
	}
	if len(op.Security) != 2 {
		t.Fatalf("security = %v, want an empty requirement plus the scheme", op.Security)
	}
	if len(op.Security[0]) != 0 {
		t.Errorf("the first requirement = %v, want empty", op.Security[0])
	}
	if _, ok := op.Security[1]["bearerAuth"]; !ok {
		t.Errorf("the second requirement = %v, want bearerAuth", op.Security[1])
	}
}

func TestDocumentRequestBody(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(Post("/x").Body(new(Payload)))

	doc := reg.Document()
	op := doc.Paths["/x"].Post
	if op.RequestBody == nil {
		t.Fatal("the request body is missing")
	}
	if !op.RequestBody.Required {
		t.Error("the request body must be required")
	}
	media, ok := op.RequestBody.Content["application/json"]
	if !ok {
		t.Fatalf("content = %v, want application/json", keys(op.RequestBody.Content))
	}
	if media.Schema == nil || media.Schema.Ref == "" {
		t.Errorf("schema = %+v, want a reference", media.Schema)
	}
}

func TestDocumentResponseContent(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(Get("/x").Resp(200, new(Payload)).Resp(204, nil))

	doc := reg.Document()
	op := doc.Paths["/x"].Get

	resp := op.Responses["200"]
	if resp == nil {
		t.Fatal("the 200 response is missing")
	}
	if _, ok := resp.Content["application/json"]; !ok {
		t.Errorf("content = %v", keys(resp.Content))
	}

	empty := op.Responses["204"]
	if empty == nil {
		t.Fatal("the 204 response is missing")
	}
	if len(empty.Content) != 0 {
		t.Errorf("content = %v, want none for a nil body", keys(empty.Content))
	}
}

func TestSortedResponsesOrdersByStatus(t *testing.T) {
	op := Get("/x").Resp(500, nil).Resp(200, nil).Resp(404, nil)

	var codes []string
	for _, resp := range sortedResponses(op.Responses) {
		codes = append(codes, resp.status)
	}

	want := []string{"200", "404", "500"}
	if len(codes) != len(want) {
		t.Fatalf("codes = %v, want %v", codes, want)
	}
	for i := range want {
		if codes[i] != want[i] {
			t.Errorf("code %d = %q, want %q", i, codes[i], want[i])
		}
	}
}

func TestStripPrefix(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(Get("/api/users"), Get("/api/posts"), Get("/other"))

	reg.stripPrefix("/api")

	paths := map[string]bool{}
	for _, op := range reg.Operations() {
		paths[op.Path] = true
	}
	if !paths["/users"] {
		t.Errorf("paths = %v, want /users", paths)
	}
	if !paths["/posts"] {
		t.Errorf("paths = %v, want /posts", paths)
	}
	if !paths["/other"] {
		t.Error("an unrelated path must not change")
	}
}

func TestStripPrefixWithEmptyPrefix(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(Get("/users"))

	reg.stripPrefix("")

	if reg.Operations()[0].Path != "/users" {
		t.Errorf("path = %q, want /users", reg.Operations()[0].Path)
	}
}

func TestStripPrefixNormalisesToRoot(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(Get("/api"))

	reg.stripPrefix("/api")

	if reg.Operations()[0].Path != "/" {
		t.Errorf("path = %q, want /", reg.Operations()[0].Path)
	}
}

func TestDocumentCoversEveryMethod(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(
		Get("/x").Resp(200, nil),
		Post("/x").Resp(200, nil),
		Put("/x").Resp(200, nil),
		Patch("/x").Resp(200, nil),
		Delete("/x").Resp(200, nil),
		Path(http.MethodOptions, "/x").Resp(200, nil),
		Path(http.MethodHead, "/x").Resp(200, nil),
		Path(http.MethodTrace, "/x").Resp(200, nil),
	)

	doc := reg.Document()
	item := doc.Paths["/x"]

	if item.Get == nil {
		t.Error("get is missing")
	}
	if item.Post == nil {
		t.Error("post is missing")
	}
	if item.Put == nil {
		t.Error("put is missing")
	}
	if item.Patch == nil {
		t.Error("patch is missing")
	}
	if item.Delete == nil {
		t.Error("delete is missing")
	}
	if item.Options == nil {
		t.Error("options is missing")
	}
	if item.Head == nil {
		t.Error("head is missing")
	}
	if item.Trace == nil {
		t.Error("trace is missing")
	}
}

func TestDocumentUsesExplicitSchemas(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(
		Get("/x").
			Param(Parameter{
				Name:   "filter",
				In:     "query",
				Schema: &spec.Schema{Type: "string"},
			}).
			Body(new(Payload)),
	)

	doc := reg.Document()
	op := doc.Paths["/x"].Get

	if len(op.Parameters) != 1 {
		t.Fatalf("parameters = %d, want 1", len(op.Parameters))
	}
	if op.Parameters[0].Schema == nil || op.Parameters[0].Schema.Type != "string" {
		t.Errorf("schema = %+v, want string", op.Parameters[0].Schema)
	}
}

func TestDocumentBodyWithNilValue(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(Post("/x").Body(nil))

	doc := reg.Document()
	op := doc.Paths["/x"].Post

	if op.RequestBody == nil {
		t.Fatal("the request body is missing")
	}
	media, ok := op.RequestBody.Content["application/json"]
	if !ok {
		t.Fatalf("content = %v", keys(op.RequestBody.Content))
	}
	if media.Schema == nil {
		t.Error("the schema is missing")
	}
}

func TestDocumentParamWithoutType(t *testing.T) {
	reg := NewRegistry("Title", "1.0.0")
	reg.Register(Get("/x").Param(Parameter{Name: "flag", In: "query"}))

	doc := reg.Document()
	op := doc.Paths["/x"].Get

	if len(op.Parameters) != 1 {
		t.Fatalf("parameters = %d, want 1", len(op.Parameters))
	}
	if op.Parameters[0].Schema == nil {
		t.Error("the schema must not be nil")
	}
}

// ---------- helpers ----------

func TestStatusKey(t *testing.T) {
	if got := statusKey(200); got != "200" {
		t.Errorf("statusKey = %q, want 200", got)
	}
	if got := statusKey(404); got != "404" {
		t.Errorf("statusKey = %q, want 404", got)
	}
}

func TestStatusText(t *testing.T) {
	if got := statusText(200); got != "OK" {
		t.Errorf("statusText = %q, want OK", got)
	}
	if got := statusText(599); got != "599" {
		t.Errorf("statusText = %q, want 599 for an unknown code", got)
	}
}

func TestTypeOf(t *testing.T) {
	if got := typeOf(nil); got != nil {
		t.Errorf("typeOf(nil) = %v, want nil", got)
	}
	if got := typeOf(Payload{}); got != reflect.TypeOf(Payload{}) {
		t.Errorf("typeOf = %v, want Payload", got)
	}
	if got := typeOf(&Payload{}); got != reflect.TypeOf(Payload{}) {
		t.Errorf("typeOf of a pointer = %v, want Payload", got)
	}
}

// keys returns the map keys as a slice.
func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
