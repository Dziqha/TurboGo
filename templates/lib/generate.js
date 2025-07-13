import fs from "fs-extra";
import path from "path";
import { execa } from "execa";

export async function generateProject(projectName, projectPath, config) {
  await fs.ensureDir(projectPath);
  const envContent = generateDotEnv();
  await fs.writeFile(path.join(projectPath, ".env"), envContent);
  const gitignoreContent = generateGitignore();
  await fs.writeFile(path.join(projectPath, ".gitignore"), gitignoreContent);
  const readmeContent = generateReadme(projectName);
  await fs.writeFile(path.join(projectPath, "README.md"), readmeContent);

  const controllerDir = path.join(projectPath, "pkg/controller");
  const routerDir = path.join(projectPath, "pkg/router");
  await fs.ensureDir(controllerDir);
  await fs.ensureDir(routerDir);

  const controllerCode = generateHandlerController(config.name);
  await fs.writeFile(
    path.join(controllerDir, `${config.name.toLowerCase()}.go`),
    controllerCode
  );

  const routerCode = generateRouter(projectName, config.name);
  await fs.writeFile(path.join(routerDir, `router.go`), routerCode);

  const mainCode = generateMainFile(projectName, config.name);
  await fs.writeFile(path.join(projectPath, "main.go"), mainCode);

  await execa("go", ["mod", "init", projectName], { cwd: projectPath });

  await execa("go", ["get", "github.com/Dziqha/TurboGo@latest"], {
    cwd: projectPath,
  });
}

// ==============================
// Generator function
// ==============================

function generateMainFile(projectName, controllerName) {
  return `package main

import (
	"github.com/Dziqha/TurboGo"
	"${projectName}/pkg/controller"
	"${projectName}/pkg/router"
)

func main() {
	app := TurboGo.New()
	ctrl := controller.New${controllerName}Controller()
	router.NewRouter(app, ctrl)
	app.RunServer(":8080")
}
`;
}

function generateHandlerController(name) {
  const lower = name.toLowerCase();
  return `package controller

import (
	"fmt"
	"github.com/Dziqha/TurboGo/core"
)

type ${name}Controller struct{}

func New${name}Controller() *${name}Controller {
	return &${name}Controller{}
}

func (h *${name}Controller) Get(c *core.Context) {
	fmt.Println("GET /${lower}")
	c.Text(200, "Hello from ${name}Controller")
}
`;
}

function generateRouter(projectName, controllerName) {
  return `package router

import (
	"github.com/Dziqha/TurboGo/core"
	"${projectName}/pkg/controller"
)

func NewRouter(router core.Router, c *controller.${controllerName}Controller) {
	app := router.Group("/api")
	app.Get("/hello", c.Get)
}
`;
}

function generateDotEnv() {
  return `PORT=8080
APP_NAME=TurboGoApp
ENV=development
`;
}

function generateGitignore() {
  return `bin/
*.exe
*.out
*.log
.env
`;
}

function generateReadme(project) {
  return `# 🚀 ${project}

Generated with [TurboGo CLI](https://github.com/username/TurboGo)

## 🚦 Jalankan:

go run .

## 📁 Struktur

- main.go — Entry point
- pkg/router — Routing logic
- pkg/controller — Handler & controller
`;
}
