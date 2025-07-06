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
	c.Text(200,"Hello from %sController")
}
`, name, name, name, name, name, name, name)
}

func GenerateRouter(module string, name string, withAuth bool) string {
	router := fmt.Sprintf(`package router

import (
	"github.com/Dziqha/TurboGo/core"
	"%s/internal/controller"
)

func NewRouter(router core.Router, c *controller.%sController) {
	app := router.Group("/api")
	app.Get("/hello", c.Get)
`, module, name)

	if withAuth {
		router += `	app.Post("/auth/login", controller.LoginHandler)
`
	}

	router += `}`
	return router
}

func GenerateAuthController() string {
	return `package controller

import (
	"github.com/Dziqha/TurboGo/core"
)

func LoginHandler(c *core.Context) {
	c.Text(200, "🔐 Login success (dummy)")
}`
}

func GenerateDotEnv() string {
	return `PORT=8080
APP_NAME=TurboGoApp
ENV=development
`
}

func GenerateGitignore() string {
	return `bin/
*.exe
*.out
*.log
.env
`
}

func GenerateReadme(project string) string {
	return fmt.Sprintf(`# 🚀 %s

Generated with [TurboGo CLI](https://github.com/username/TurboGo)

## 🚦 Jalankan:

go run .

## 📁 Struktur

- main.go — Entry point
- internal/router — Routing logic
- internal/controller — Handler & controller
`, project)
}
