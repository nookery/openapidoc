package serve

import (
	"github.com/nookery/openapidoc/api"
)

// newTestRegistry builds a small registry for the tests.
func newTestRegistry() *api.Registry {
	reg := api.NewRegistry("Test API", "1.0.0")
	reg.Register(
		api.Get("/users/{id}").
			Summary("Get a user").
			PathParam("id", 0).
			Resp(200, new(struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			})),
	)
	return reg
}
