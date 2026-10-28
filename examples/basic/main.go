// Command example runs a small service that serves its own API
// documentation.
//
// Run it with:
//
//	go run ./examples/basic
//
// Then open http://localhost:8080/docs in a browser.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/nookery/openapidoc"
)

// User is a resource of the example API.
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name" jsonschema:"description=Display name of the user"`
	Email     string    `json:"email" jsonschema:"description=Contact address"`
	Role      string    `json:"role" jsonschema:"description=Role of the user,enum=admin|member|guest"`
	Age       *int      `json:"age,omitempty" jsonschema:"description=Age in years,minimum=0,maximum=150"`
	CreatedAt time.Time `json:"created_at" jsonschema:"description=Moment the user was created"`
}

// CreateUserRequest is the payload accepted when creating a user.
type CreateUserRequest struct {
	Name  string `json:"name" jsonschema:"description=Display name of the user,required"`
	Email string `json:"email" jsonschema:"description=Contact address,required"`
	Role  string `json:"role,omitempty" jsonschema:"description=Role of the user,enum=admin|member|guest"`
}

// ErrorResponse is returned when a request fails.
type ErrorResponse struct {
	Code    int    `json:"code" jsonschema:"description=Machine readable error code"`
	Message string `json:"message" jsonschema:"description=Human readable error message"`
}

func main() {
	// Describe the API. Every operation is declared next to the data it
	// accepts and returns, so the documentation cannot drift from the code.
	docs := openapidoc.New("Example API", "1.0.0").
		Description("A small API that demonstrates openapidoc.")
	docs.AddServer("http://localhost:8080", "Local development")
	docs.AddTag("users", "User management")
	docs.BearerAuth("bearerAuth", "JWT access token")

	docs.Register(
		openapidoc.Get("/api/users").
			Summary("List users").
			Describe("Returns a paginated list of users.").
			Tag("users").
			Query("page", 1).
			Query("page_size", 20).
			Security("bearerAuth").
			Resp(200, new([]User)).
			Resp(401, new(ErrorResponse)),

		openapidoc.Get("/api/users/{id}").
			Summary("Get a user").
			Describe("Returns a single user by its identifier.").
			Tag("users").
			PathParam("id", 0).
			Security("bearerAuth").
			Resp(200, new(User)).
			Resp(404, new(ErrorResponse)),

		openapidoc.Post("/api/users").
			Summary("Create a user").
			Tag("users").
			Body(new(CreateUserRequest)).
			Resp(201, new(User)).
			Resp(400, new(ErrorResponse)),

		openapidoc.Delete("/api/users/{id}").
			Summary("Delete a user").
			Tag("users").
			PathParam("id", 0).
			Security("bearerAuth").
			Resp(204, nil).
			Resp(404, new(ErrorResponse)),
	)

	// Mount the documentation. The handler serves both the UI and the
	// OpenAPI document.
	handler := openapidoc.NewHandler(docs)

	mux := http.NewServeMux()
	mux.Handle(handler.DocsPath()+"/", handler.DocsHandler())
	mux.Handle(handler.SpecPath(), handler.SpecHandler())

	// A real endpoint, so that the try it panel has something to call.
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]User{{
			ID:        1,
			Name:      "Ada Lovelace",
			Email:     "ada@example.com",
			Role:      "admin",
			CreatedAt: time.Now(),
		}})
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Println("API documentation:  http://localhost:8080/docs")
	fmt.Println("OpenAPI document:   http://localhost:8080/docs/openapi.json")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
