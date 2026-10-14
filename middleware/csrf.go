package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"time"

	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

type CSRFOptions struct {
	CookieName string
	HeaderName string
	TokenLen   int
}

var csrfSafeMethods = map[string]bool{
	"GET":     true,
	"HEAD":    true,
	"OPTIONS": true,
}

func CSRF(opts ...CSRFOptions) core.Handler {
	o := CSRFOptions{
		CookieName: "csrf_token",
		HeaderName: "X-CSRF-Token",
		TokenLen:   32,
	}
	if len(opts) > 0 {
		if opts[0].CookieName != "" {
			o.CookieName = opts[0].CookieName
		}
		if opts[0].HeaderName != "" {
			o.HeaderName = opts[0].HeaderName
		}
		if opts[0].TokenLen > 0 {
			o.TokenLen = opts[0].TokenLen
		}
	}

	return func(c *core.Context) {
		if csrfSafeMethods[string(c.Ctx.Method())] {
			ensureCSRFToken(c, o)
			c.Next()
			return
		}

		cookie := string(c.Ctx.Request.Header.Cookie(o.CookieName))
		header := string(c.Ctx.Request.Header.Peek(o.HeaderName))
		if cookie == "" || header == "" ||
			subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) != 1 {
			c.Ctx.SetStatusCode(fasthttp.StatusForbidden)
			c.Ctx.SetBodyString("invalid csrf token")
			c.Abort()
			return
		}

		c.Next()
	}
}

func ensureCSRFToken(c *core.Context, o CSRFOptions) {
	if len(c.Ctx.Request.Header.Cookie(o.CookieName)) > 0 {
		return
	}

	b := make([]byte, o.TokenLen)
	if _, err := rand.Read(b); err != nil {
		return
	}

	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)

	cookie.SetKey(o.CookieName)
	cookie.SetValue(hex.EncodeToString(b))
	cookie.SetPath("/")
	cookie.SetHTTPOnly(false)
	cookie.SetMaxAge(int((24 * time.Hour).Seconds()))
	c.SetCookie(cookie)
}