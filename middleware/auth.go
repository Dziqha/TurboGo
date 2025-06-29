// auth.go
package middleware

import (
	"strings"
	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

func Auth() core.Handler {
	return func(c *core.Context) {
		authHeader := string(c.Ctx.Request.Header.Peek("Authorization"))
		
		// Check if Authorization header exists and has Bearer prefix
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			c.Ctx.SetContentType("application/json")
			c.Ctx.SetBodyString(`{"error":"unauthorized","message":"missing or invalid authorization header"}`)
			return
		}
		
		// Extract token (remove "Bearer " prefix)
		token := strings.TrimPrefix(authHeader, "Bearer ")
		
		if token != "mysecrettoken" {
			c.Ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			c.Ctx.SetContentType("application/json")
			c.Ctx.SetBodyString(`{"error":"unauthorized","message":"invalid token"}`)
			return
		}
		
		c.Next()
	}
}