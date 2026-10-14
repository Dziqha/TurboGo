package middleware

import (
	"crypto/subtle"
	"encoding/base64"
	"strings"

	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

type BasicAuthUser struct {
	Username string
	Password string
}

type BasicAuthOptions struct {
	Realm string
}

func BasicAuth(users []BasicAuthUser, opts ...BasicAuthOptions) core.Handler {
	return BasicAuthWithValidator(func(username, password string) bool {
		for _, u := range users {
			if subtle.ConstantTimeCompare([]byte(username), []byte(u.Username)) == 1 &&
				subtle.ConstantTimeCompare([]byte(password), []byte(u.Password)) == 1 {
				return true
			}
		}
		return false
	}, opts...)
}

func BasicAuthWithValidator(validate func(username, password string) bool, opts ...BasicAuthOptions) core.Handler {
	realm := "Restricted"
	if len(opts) > 0 && opts[0].Realm != "" {
		realm = opts[0].Realm
	}

	return func(c *core.Context) {
		auth := string(c.Ctx.Request.Header.Peek("Authorization"))
		if !strings.HasPrefix(auth, "Basic ") {
			basicAuthUnauthorized(c, realm)
			return
		}

		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		if err != nil {
			basicAuthUnauthorized(c, realm)
			return
		}

		username, password, ok := strings.Cut(string(raw), ":")
		if !ok {
			basicAuthUnauthorized(c, realm)
			return
		}

		if !validate(username, password) {
			basicAuthUnauthorized(c, realm)
			return
		}

		c.SetSession("username", username)
		c.SetSession("password", password)
		c.Next()
	}
}

func basicAuthUnauthorized(c *core.Context, realm string) {
	c.Ctx.Response.Header.Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
	c.Ctx.SetStatusCode(fasthttp.StatusUnauthorized)
	c.Ctx.SetBodyString("unauthorized")
	c.Abort()
}