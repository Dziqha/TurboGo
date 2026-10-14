package middleware

import (
	"strconv"
	"strings"

	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func CORS(opts ...CORSOptions) core.Handler {
	o := CORSOptions{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: false,
		MaxAge:           86400,
	}
	if len(opts) > 0 {
		if len(opts[0].AllowedOrigins) > 0 {
			o.AllowedOrigins = opts[0].AllowedOrigins
		}
		if len(opts[0].AllowedMethods) > 0 {
			o.AllowedMethods = opts[0].AllowedMethods
		}
		if len(opts[0].AllowedHeaders) > 0 {
			o.AllowedHeaders = opts[0].AllowedHeaders
		}
		if len(opts[0].ExposedHeaders) > 0 {
			o.ExposedHeaders = opts[0].ExposedHeaders
		}
		if opts[0].AllowCredentials {
			o.AllowCredentials = true
		}
		if opts[0].MaxAge > 0 {
			o.MaxAge = opts[0].MaxAge
		}
	}

	allowAllOrigins := false
	for _, origin := range o.AllowedOrigins {
		if origin == "*" {
			allowAllOrigins = true
			break
		}
	}

	methods := strings.Join(o.AllowedMethods, ", ")
	headers := strings.Join(o.AllowedHeaders, ", ")
	exposed := strings.Join(o.ExposedHeaders, ", ")
	maxAge := strconv.Itoa(o.MaxAge)

	return func(c *core.Context) {
		origin := string(c.Ctx.Request.Header.Peek("Origin"))
		originAllowed := allowAllOrigins || containsStr(o.AllowedOrigins, origin)

		if origin != "" && originAllowed {
			if allowAllOrigins && !o.AllowCredentials {
				c.Ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
			} else {
				c.Ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
				c.Ctx.Response.Header.Set("Vary", "Origin")
			}
			if o.AllowCredentials {
				c.Ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
			}
			if exposed != "" {
				c.Ctx.Response.Header.Set("Access-Control-Expose-Headers", exposed)
			}
		}

		if string(c.Ctx.Method()) == "OPTIONS" && origin != "" {
			if originAllowed {
				c.Ctx.Response.Header.Set("Access-Control-Allow-Methods", methods)
				reqHeaders := string(c.Ctx.Request.Header.Peek("Access-Control-Request-Headers"))
				if reqHeaders == "" {
					reqHeaders = headers
				}
				if reqHeaders != "" {
					c.Ctx.Response.Header.Set("Access-Control-Allow-Headers", reqHeaders)
				}
				if o.MaxAge > 0 {
					c.Ctx.Response.Header.Set("Access-Control-Max-Age", maxAge)
				}
			}
			c.Ctx.SetStatusCode(fasthttp.StatusNoContent)
			c.Abort()
			return
		}

		c.Next()
	}
}

func containsStr(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}