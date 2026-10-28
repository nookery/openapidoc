# openapidoc

English | [简体中文](README.zh-CN.md)

Browser readable API documentation for Go services. Describe the API once,
mount the handler, open the browser.

No code generation. No `swag init`. No build step. No comments to keep in
sync. The OpenAPI document is built at run time from the Go types you already
have.

```go
docs := openapidoc.New("My API", "1.0.0")

docs.Register(
	openapidoc.Get("/users/{id}").
		Summary("Get a user").
		PathParam("id", 0).
		Resp(200, new(User)),
)

handler := openapidoc.NewHandler(docs)
mux := http.NewServeMux()
mux.Handle(handler.DocsPath()+"/", handler.DocsHandler())
mux.Handle(handler.SpecPath(), handler.SpecHandler())

http.ListenAndServe(":8080", mux)
```

The documentation is now at `http://localhost:8080/docs`.

## Install

```bash
go get github.com/nookery/openapidoc
```

The package depends only on the Go standard library, so it adds nothing to
your dependency tree and compiles into any router.

## Why

The usual way to get API documentation in Go is a code generator:

| | Generator based | openapidoc |
|---|---|---|
| Extra build step | yes | no |
| Generated files in the repo | yes | no |
| Drifts from the code | silently | cannot |
| Works without the source | no | yes |
| Output | often Swagger 2.0 | OpenAPI 3.1 |

A generator parses your source and writes a document. The document is a
snapshot, and nothing reminds you to regenerate it. openapidoc reflects on
the Go types at run time, so the document is derived from the code that is
actually running.

## How it works

Pass a Go value to describe a type. The package reflects on it and produces
a JSON Schema.

```go
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name" jsonschema:"description=Display name"`
	Email string `json:"email,omitempty"`
}
```

- Pointer fields and `omitempty` are optional, everything else is required.
- Named struct types become reusable components referenced with `$ref`.
- Recursive types terminate, a self referencing field points back at its own
  component.
- `time.Time` becomes `string` with format `date-time`.

### Struct tags

The `jsonschema` tag adjusts the generated schema:

```go
type Product struct {
	Price  int    `json:"price" jsonschema:"description=Price in cents,minimum=0"`
	Tags   string `json:"tags" jsonschema:"description=Comma separated tags,pattern=^[a-z,]+$"`
	Kind   string `json:"kind" jsonschema:"enum=book|disc|digital"`
	Note   string `json:"note,omitempty" jsonschema:"maxLength=500"`
}
```

Supported keys: `description`, `title`, `format`, `pattern`, `default`,
`example`, `enum`, `minimum`, `maximum`, `exclusiveMinimum`,
`exclusiveMaximum`, `minLength`, `maxLength`, `minItems`, `maxItems`.

## Declaring an API

### Operations

```go
openapidoc.Post("/users").
	Summary("Create a user").
	Describe("Creates a user and returns it.").
	Tag("users").
	ID("createUser").
	Body(new(CreateUserRequest)).
	Resp(201, new(User)).
	Resp(400, new(ErrorResponse))
```

### Parameters

```go
openapidoc.Get("/users").
	RequiredQuery("email", "").
	Query("page", 1).
	Header("X-Request-ID", "").
	PathParam("id", 0)
```

Path parameters are always required.

### Authentication

Register a scheme once, then require it on the operations that need it.

```go
docs.BearerAuth("bearerAuth", "JWT access token")
docs.APIKeyAuth("apiKey", "X-API-Key", "API key")

openapidoc.Get("/me").
	Security("bearerAuth")
```

Use `SecurityOptional` for endpoints that work with or without credentials.

### Document level metadata

```go
docs := openapidoc.New("My API", "1.0.0").
	Description("Everything the public API can do.").
	AddServer("https://api.example.com", "Production").
	AddServer("http://localhost:8080", "Local development").
	AddTag("users", "User management")
```

## Serving

```go
handler := openapidoc.NewHandler(docs, openapidoc.Options{
	DocsPath: "/api-docs",
})
```

The handler exposes two routes:

- `handler.DocsPath()` serves the documentation UI
- `handler.SpecPath()` serves the OpenAPI document as JSON

Both are plain `http.Handler` values, so they mount on any router.

```go
// net/http
mux.Handle(handler.DocsPath()+"/", handler.DocsHandler())
mux.Handle(handler.SpecPath(), handler.SpecHandler())

// gin
r.GET(handler.DocsPath()+"/*any", gin.WrapH(handler.DocsHandler()))
r.GET(handler.SpecPath(), gin.WrapH(handler.SpecHandler()))
```

## Choosing a UI

The default UI is bundled with the package. It is written in plain HTML,
CSS and JavaScript with no framework and no external dependency, and is
embedded in the binary, so the documentation works offline.

```go
handler := openapidoc.NewHandler(docs)
```

It provides a three column layout: grouped navigation with search on the
left, the operation detail in the middle, and code samples on the right. The
detail view lets you fill in parameters, inspect the schema tree and send a
real request.

A custom interface is a single method:

```go
type UI interface {
	Serve(w http.ResponseWriter, r *http.Request, ctx serve.UIContext)
}
```

The package also ships interfaces backed by third party renderers:

```go
handler.WithUI(serve.ScalarUI())
handler.WithUI(serve.RedocUI())
```

Those load their assets from a CDN, so they need network access.

## Status

Early. The API surface may still change.

## License

MIT
