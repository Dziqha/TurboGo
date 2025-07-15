package middleware

import (
	"fmt"
	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
	"runtime/debug"
)

func Recover() core.Handler {
	return func(c *core.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic with stack trace
				fmt.Printf("[RECOVER] Panic recovered: %v\n", r)
				fmt.Printf("[RECOVER] Stack trace:\n%s\n", debug.Stack())

				// Set error response
				c.Ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				c.Ctx.SetContentType("application/json")
				c.Ctx.SetBodyString(`{"error":"internal server error","message":"an unexpected error occurred"}`)
			}
		}()

		c.Next()
	}
}
