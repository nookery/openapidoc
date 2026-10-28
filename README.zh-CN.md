# openapidoc

[English](README.md) | 简体中文

为 Go 服务提供浏览器可读的 API 文档。描述一次接口，挂载 handler，打开浏览器即可。

无需代码生成。无需 `swag init`。无需额外构建步骤。无需维护注释。OpenAPI 文档在运行时由你已有的 Go 类型直接生成。

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

文档地址为 `http://localhost:8080/docs`。

## 安装

```bash
go get github.com/nookery/openapidoc
```

本包只依赖 Go 标准库，不会给你的依赖树增加任何第三方库，可以编译进任意路由器。

## 为什么

Go 里做 API 文档的常规方式是代码生成器：

| | 生成器方案 | openapidoc |
|---|---|---|
| 额外的构建步骤 | 需要 | 不需要 |
| 仓库中残留生成文件 | 会 | 不会 |
| 与代码脱节 | 静默发生 | 不可能发生 |
| 没有源码时可用 | 不可用 | 可用 |
| 输出格式 | 常见为 Swagger 2.0 | OpenAPI 3.1 |

生成器解析源码并写出一份文档。那份文档只是一张快照，没有任何机制提醒你重新生成。openapidoc 则在运行时反射 Go 类型，因此文档直接派生自真正在运行的代码。

## 工作原理

传入一个 Go 值来描述其类型，本包通过反射生成对应的 JSON Schema。

```go
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name" jsonschema:"description=Display name"`
	Email string `json:"email,omitempty"`
}
```

- 指针字段与带 `omitempty` 的字段为可选，其余为必填。
- 具名结构体类型会成为可复用的 component，通过 `$ref` 引用。
- 递归类型可以正确终止，自引用字段会指回自身的 component。
- `time.Time` 转换为 `string`，format 为 `date-time`。

### 结构体标签

`jsonschema` 标签用于微调生成的 schema：

```go
type Product struct {
	Price  int    `json:"price" jsonschema:"description=Price in cents,minimum=0"`
	Tags   string `json:"tags" jsonschema:"description=Comma separated tags,pattern=^[a-z,]+$"`
	Kind   string `json:"kind" jsonschema:"enum=book|disc|digital"`
	Note   string `json:"note,omitempty" jsonschema:"maxLength=500"`
}
```

支持的键：`description`、`title`、`format`、`pattern`、`default`、`example`、`enum`、`minimum`、`maximum`、`exclusiveMinimum`、`exclusiveMaximum`、`minLength`、`maxLength`、`minItems`、`maxItems`。

## 声明接口

### 操作

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

### 参数

```go
openapidoc.Get("/users").
	RequiredQuery("email", "").
	Query("page", 1).
	Header("X-Request-ID", "").
	PathParam("id", 0)
```

路径参数始终为必填。

### 认证

先注册一次认证方案，再在需要它的操作上声明。

```go
docs.BearerAuth("bearerAuth", "JWT access token")
docs.APIKeyAuth("apiKey", "X-API-Key", "API key")

openapidoc.Get("/me").
	Security("bearerAuth")
```

对于允许匿名访问的接口，使用 `SecurityOptional`。

### 文档级元信息

```go
docs := openapidoc.New("My API", "1.0.0").
	Description("Everything the public API can do.").
	AddServer("https://api.example.com", "Production").
	AddServer("http://localhost:8080", "Local development").
	AddTag("users", "User management")
```

## 挂载服务

```go
handler := openapidoc.NewHandler(docs, openapidoc.Options{
	DocsPath: "/api-docs",
})
```

handler 暴露两个路由：

- `handler.DocsPath()` 提供文档 UI
- `handler.SpecPath()` 以 JSON 形式提供 OpenAPI 文档

两者都是标准的 `http.Handler`，因此可以挂载到任意路由器上。

```go
// net/http
mux.Handle(handler.DocsPath()+"/", handler.DocsHandler())
mux.Handle(handler.SpecPath(), handler.SpecHandler())

// gin
r.GET(handler.DocsPath()+"/*any", gin.WrapH(handler.DocsHandler()))
r.GET(handler.SpecPath(), gin.WrapH(handler.SpecHandler()))
```

## 选择 UI

默认使用 [Scalar](https://scalar.com)，它渲染出带内置 API 客户端的交互式文档。

```go
handler := openapidoc.NewHandler(docs)
handler.WithUI(serve.RedocUI())
```

自定义 UI 只需实现一个方法：

```go
type UI interface {
	Serve(w http.ResponseWriter, r *http.Request, ctx serve.UIContext)
}
```

## 状态

项目处于早期阶段，API 表面仍可能变化。

## 许可证

MIT
