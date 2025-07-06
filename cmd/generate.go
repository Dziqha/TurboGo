package main

import "fmt"

func GenerateMainFile(module string, name string) string {
	return fmt.Sprintf(`package main

import (
	"github.com/Dziqha/TurboGo"
	"%s/internal/controller"
	"%s/internal/router"
)

func main() {
	app := TurboGo.New()
	ctrl := controller.New%sController()
	router.NewRouter(app, ctrl)
	app.RunServer(":8080")
}
`, module, module, name)
}



func GenerateHandlerController(name string) string {
	return fmt.Sprintf(`package controller

import (
	"fmt"
	"github.com/Dziqha/TurboGo/core"
)

type %sController struct{}

func New%sController() *%sController {
	return &%sController{}
}

func (h *%sController) Get(c *core.Context) {
	fmt.Println("GET /%s")
	c.Text("Hello from %sController")
}
`, name, name, name, name, name, name, name)

}

func GenerateRouter(module string, name string) string {
	return fmt.Sprintf(`package router

import (
	"github.com/Dziqha/TurboGo/core"
	"%s/internal/controller"
)

func NewRouter(router core.Router, c *controller.%sController) {
	app := router.Group("/api")
	app.Get("/hello", c.Get)
}
`, module, name)
}

