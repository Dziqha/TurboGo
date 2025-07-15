package route

import (
	"github.com/Dziqha/TurboGo/core"
	"github.com/Dziqha/TurboGo/examples/controller"
)

func NewRouter(router core.Router, controller *controller.HandlerController) {
	app := router.Group("/api")
	app.Get("/", controller.GetHandler)
}
