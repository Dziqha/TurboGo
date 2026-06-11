package middleware

import (
	"testing"

	"github.com/Dziqha/TurboGo/core"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestRecover_NoPanic_PassesThrough(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := core.NewContext(&fctx, nil, nil)
	called := false

	h := core.Handler(func(c *core.Context) {
		called = true
	})

	c.SetHandlers([]core.Handler{Recover(), h})
	c.Next()

	assert.True(t, called)
}

func TestRecover_Panic_Returns500(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := core.NewContext(&fctx, nil, nil)

	h := core.Handler(func(c *core.Context) {
		panic("test panic")
	})

	c.SetHandlers([]core.Handler{Recover(), h})
	c.Next()

	assert.Equal(t, 500, fctx.Response.StatusCode())
	assert.Contains(t, string(fctx.Response.Body()), "internal server error")
}

func TestRecover_Panic_DoesNotAffectNextRequest(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := core.NewContext(&fctx, nil, nil)

	h := core.Handler(func(c *core.Context) {
		panic("panic")
	})

	c.SetHandlers([]core.Handler{Recover(), h})
	c.Next()

	var fctx2 fasthttp.RequestCtx
	c2 := core.NewContext(&fctx2, nil, nil)
	called := false
	h2 := core.Handler(func(c *core.Context) {
		called = true
	})
	c2.SetHandlers([]core.Handler{Recover(), h2})
	c2.Next()

	assert.True(t, called)
}

func TestAuthJWT_MissingHeader(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := core.NewContext(&fctx, nil, nil)

	c.SetHandlers([]core.Handler{AuthJWT("secret")})
	c.Next()

	assert.Equal(t, 401, fctx.Response.StatusCode())
	assert.True(t, c.Aborted())
}

func TestAuthJWT_InvalidScheme(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("Authorization", "Basic token")
	c := core.NewContext(&fctx, nil, nil)

	c.SetHandlers([]core.Handler{AuthJWT("secret")})
	c.Next()

	assert.Equal(t, 401, fctx.Response.StatusCode())
	assert.True(t, c.Aborted())
}

func TestAuthJWT_InvalidToken(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("Authorization", "Bearer invalidtoken")
	c := core.NewContext(&fctx, nil, nil)

	c.SetHandlers([]core.Handler{AuthJWT("secret")})
	c.Next()

	assert.Equal(t, 401, fctx.Response.StatusCode())
	assert.True(t, c.Aborted())
}

func TestAuthJWT_ValidToken_WithClaims(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user123",
	})
	tokenStr, err := token.SignedString([]byte("mysecret"))
	assert.NoError(t, err)

	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("Authorization", "Bearer "+tokenStr)
	c := core.NewContext(&fctx, nil, nil)

	called := false
	h := core.Handler(func(c *core.Context) {
		called = true
		assert.Equal(t, "user123", c.GetSession("user"))
	})

	c.SetHandlers([]core.Handler{AuthJWT("mysecret"), h})
	c.Next()

	assert.True(t, called)
}

func TestAuthJWT_WrongSecret(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user123",
	})
	tokenStr, err := token.SignedString([]byte("correctsecret"))
	assert.NoError(t, err)

	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("Authorization", "Bearer "+tokenStr)
	c := core.NewContext(&fctx, nil, nil)

	c.SetHandlers([]core.Handler{AuthJWT("wrongsecret")})
	c.Next()

	assert.Equal(t, 401, fctx.Response.StatusCode())
	assert.True(t, c.Aborted())
}

func TestAuthJWT_ExpiredToken(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": 1, // expired in 1970
	})
	tokenStr, err := token.SignedString([]byte("secret"))
	assert.NoError(t, err)

	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("Authorization", "Bearer "+tokenStr)
	c := core.NewContext(&fctx, nil, nil)

	c.SetHandlers([]core.Handler{AuthJWT("secret")})
	c.Next()

	assert.Equal(t, 401, fctx.Response.StatusCode())
	assert.True(t, c.Aborted())
}
