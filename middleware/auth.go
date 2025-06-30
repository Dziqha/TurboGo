package middleware

import (
	"strings"
	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

func Auth(secret string) core.Handler {
	return func(c *core.Context) {
		authHeader := string(c.Ctx.Request.Header.Peek("Authorization"))
		
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			c.Ctx.SetContentType("application/json")
			c.Ctx.SetBodyString(`{"error":"unauthorized","message":"missing or invalid authorization header"}`)
			return
		}
		
		token := strings.TrimPrefix(authHeader, "Bearer ")
		
		if token != secret {
			c.Ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			c.Ctx.SetContentType("application/json")
			c.Ctx.SetBodyString(`{"error":"unauthorized","message":"invalid token"}`)
			return
		}
		
		c.Next()
	}
}