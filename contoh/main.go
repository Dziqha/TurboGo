package main

import (
	"encoding/json"
	"fmt"


	"github.com/valyala/fasthttp"
)

type Context struct {
	Ctx      *fasthttp.RequestCtx
	Response []byte
}

type Handler func(*Context)

var memoryCache = map[string][]byte{}

func JSONHandler(c *Context, body any) {
	c.Ctx.SetStatusCode(200)
	c.Ctx.Response.Header.SetContentType("application/json")
	res, _ := json.Marshal(body)
	c.Ctx.SetBody(res)
}

func CacheMiddleware(path string, handler Handler) Handler {
	return func(c *Context) {
		if val, ok := memoryCache[path]; ok {
			c.Ctx.SetStatusCode(200)
			c.Ctx.Response.Header.SetContentType("application/json")
			c.Ctx.SetBody(val)
			fmt.Println("✅ FROM CACHE")
			return
		}

		handler(c)

		body := c.Ctx.Response.Body()
		if len(body) > 0 {
			memoryCache[path] = append([]byte(nil), body...)
			fmt.Println("💾 STORED TO CACHE:", string(body))
		}
	}
}

func mainHandler(ctx *fasthttp.RequestCtx) {
	c := &Context{Ctx: ctx}

	switch string(ctx.Path()) {
	case "/public":
		handler := CacheMiddleware("/public", func(c *Context) {
			fmt.Println("🔥 PublicHandler executed")
			JSONHandler(c, map[string]any{"message": "this is public content"})
		})
		handler(c)

	default:
		ctx.SetStatusCode(404)
		ctx.SetBodyString("Not Found")
	}
}

func main() {
	fmt.Println("Listening on :8080")
	fasthttp.ListenAndServe(":8080", mainHandler)
}
