package rest

import (
	"testing"

	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/rest/api"
	"github.com/tencent-go/pkg/rest/router"
)

func TestRest(t *testing.T) {
	type Input struct {
		Value string `json:"value"`
	}

	type Output struct {
		Value string `json:"value"`
	}

	api1 := api.NewEndpoint[Input, Output]().WithPath("api1").WithName("api1").WithMethod(api.MethodGet)
	api2 := api.NewEndpoint[Input, Output]().WithPath("api2").WithName("api2").WithMethod(api.MethodPost)
	group := api.DefaultGroup().WithPath("api/v1").WithName("business").WithChildren(api1, api2)

	r := router.NewWithDefaultMiddlewares()
	r.AddNodes(group)
	api.PrintRoutes(r.GetRoutes())

	t.Run("register handlers", func(t *testing.T) {
		router.RegisterEndpointHandler(r, api1, func(ctx router.Context, params Input) (*Output, errx.Error) {
			return &Output{params.Value}, nil
		})
		router.RegisterEndpointHandler(r, api2, func(ctx router.Context, params Input) (*Output, errx.Error) {
			return &Output{params.Value}, nil
		})
	})

	t.Run("run", func(t *testing.T) {
		_ = r.RunH2C(":80")
	})
}
