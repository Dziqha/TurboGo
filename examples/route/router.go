package route

import (
	"github.com/Dziqha/TurboGo"
	"github.com/Dziqha/TurboGo/core"
	"github.com/Dziqha/TurboGo/examples/controller"
)

func NewRouter(router *TurboGo.App, controller *controller.HandlerController) {
	router.Get("/:name", controller.GetHandler)
	router.Post("/hello", func(ctx *core.Context) {
		ctx.Text(200, "Hello World")
	})
}
