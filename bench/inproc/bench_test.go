// Package inproc measures framework overhead with no network in the loop.
//
// Every case is called as a fasthttp.RequestHandler with a reused
// RequestCtx, so the number is "what the framework adds on top of
// fasthttp's parser/writer" — the layer a wrk run cannot isolate (under
// load, syscall time dominates and router cost disappears in noise).
package inproc

import (
	"testing"

	"github.com/Dziqha/TurboGo"
	"github.com/Dziqha/TurboGo/core"
	fiber2 "github.com/gofiber/fiber/v2"
	fiber3 "github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

const paramPath = "/user/42"

// callLoop mimics the server loop: reset request/response, set the line,
// invoke the handler.
func callLoop(b *testing.B, h fasthttp.RequestHandler, path string) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&fasthttp.Request{}, nil, nil)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI(path)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.Reset()
		ctx.Request.Reset()
		ctx.Request.Header.SetMethod("GET")
		ctx.Request.SetRequestURI(path)
		h(ctx)
	}
}

func BenchmarkBare(b *testing.B) {
	b.Run("fasthttp", func(b *testing.B) {
		h := func(ctx *fasthttp.RequestCtx) {
			p := ctx.Path()
			ctx.SetBodyString("user:" + string(p[6:]))
		}
		callLoop(b, h, paramPath)
	})

	b.Run("fasthttp-static", func(b *testing.B) {
		h := func(ctx *fasthttp.RequestCtx) {
			ctx.SetBodyString("Hello, World!")
		}
		callLoop(b, h, "/plaintext")
	})
}

func BenchmarkTurboGo(b *testing.B) {
	b.Run("static", func(b *testing.B) {
		app := TurboGo.New()
		app.Get("/plaintext", func(c *core.Context) {
			c.Ctx.SetContentType("text/plain")
			c.Ctx.SetBodyString("Hello, World!")
		})
		callLoop(b, app.Handler(), "/plaintext")
	})

	b.Run("param", func(b *testing.B) {
		app := TurboGo.New()
		app.Get("/user/:id", func(c *core.Context) {
			c.Ctx.SetContentType("text/plain")
			c.Ctx.SetBodyString("user:" + c.Param("id"))
		})
		callLoop(b, app.Handler(), paramPath)
	})

	b.Run("fast", func(b *testing.B) {
		app := TurboGo.New()
		app.GetFast(paramPath, func(ctx *fasthttp.RequestCtx) {
			ctx.SetBodyString("user:42")
		})
		callLoop(b, app.Handler(), paramPath)
	})

	b.Run("param-json", func(b *testing.B) {
		app := TurboGo.New()
		app.Get("/user/:id", func(c *core.Context) {
			c.JSON(200, map[string]string{"id": c.Param("id")})
		})
		callLoop(b, app.Handler(), paramPath)
	})
}

func BenchmarkFiber(b *testing.B) {
	b.Run("v2", func(b *testing.B) {
		app := fiber2.New(fiber2.Config{DisableStartupMessage: true})
		app.Get("/user/:id", func(c *fiber2.Ctx) error {
			return c.SendString("user:" + c.Params("id"))
		})
		callLoop(b, app.Handler(), paramPath)
	})

	b.Run("v3", func(b *testing.B) {
		app := fiber3.New()
		app.Get("/user/:id", func(c fiber3.Ctx) error {
			return c.SendString("user:" + c.Params("id"))
		})
		callLoop(b, app.Handler(), paramPath)
	})
}
