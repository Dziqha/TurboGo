package inproc

import (
	"testing"

	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

func BenchmarkOverhead(b *testing.B) {
	const path = "/plaintext"

	b.Run("bare", func(b *testing.B) {
		callLoop(b, func(ctx *fasthttp.RequestCtx) { ctx.SetBodyString("Hello, World!") }, path)
	})

	b.Run("bare+recover", func(b *testing.B) {
		h := func(ctx *fasthttp.RequestCtx) {
			defer func() { _ = recover() }()
			ctx.SetBodyString("Hello, World!")
		}
		callLoop(b, h, path)
	})

	b.Run("bare+ctx-lifecycle", func(b *testing.B) {
		h := func(ctx *fasthttp.RequestCtx) {
			c := core.NewContext(ctx, nil, nil)
			defer core.ReleaseContext(c)
			ctx.SetBodyString("Hello, World!")
		}
		callLoop(b, h, path)
	})

	b.Run("bare+ctx+recover", func(b *testing.B) {
		h := func(ctx *fasthttp.RequestCtx) {
			c := core.NewContext(ctx, nil, nil)
			defer core.ReleaseContext(c)
			defer func() { _ = recover() }()
			ctx.SetBodyString("Hello, World!")
		}
		callLoop(b, h, path)
	})

	b.Run("ctx-only", func(b *testing.B) {
		h := func(ctx *fasthttp.RequestCtx) {
			c := core.NewContext(ctx, nil, nil)
			c.Ctx.SetContentType("text/plain")
			c.Ctx.SetBodyString("Hello, World!")
			core.ReleaseContext(c)
		}
		callLoop(b, h, path)
	})
}
