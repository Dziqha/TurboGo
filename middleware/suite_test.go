package middleware

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

func basicAuthEncode(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

// ===== CORS =====

func TestCORS_Preflight(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetMethod("OPTIONS")
	fctx.Request.Header.Set("Origin", "https://example.com")
	c := core.NewContext(&fctx, nil, nil)

	h := core.Handler(func(c *core.Context) {
		t.Error("preflight request should not reach handler")
	})

	c.SetHandlers([]core.Handler{CORS(), h})
	c.Next()

	if fctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("expected 204, got %d", fctx.Response.StatusCode())
	}
	if !c.Aborted() {
		t.Fatal("expected context to be aborted")
	}
	if got := string(fctx.Response.Header.Peek("Access-Control-Allow-Origin")); got != "*" {
		t.Fatalf("expected Allow-Origin *, got %q", got)
	}
	if got := string(fctx.Response.Header.Peek("Access-Control-Allow-Methods")); !strings.Contains(got, "POST") {
		t.Fatalf("expected POST in Allow-Methods, got %q", got)
	}
}

func TestCORS_EchoOriginWithCredentials(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetMethod("GET")
	fctx.Request.Header.Set("Origin", "https://example.com")
	c := core.NewContext(&fctx, nil, nil)

	nextCalled := false
	h := core.Handler(func(c *core.Context) {
		nextCalled = true
	})

	mw := CORS(CORSOptions{
		AllowedOrigins:   []string{"https://example.com"},
		AllowCredentials: true,
	})
	c.SetHandlers([]core.Handler{mw, h})
	c.Next()

	if !nextCalled {
		t.Fatal("handler should be called")
	}
	if got := string(fctx.Response.Header.Peek("Access-Control-Allow-Origin")); got != "https://example.com" {
		t.Fatalf("expected origin echo, got %q", got)
	}
	if got := string(fctx.Response.Header.Peek("Access-Control-Allow-Credentials")); got != "true" {
		t.Fatalf("expected Allow-Credentials true, got %q", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetMethod("GET")
	fctx.Request.Header.Set("Origin", "https://evil.com")
	c := core.NewContext(&fctx, nil, nil)

	nextCalled := false
	h := core.Handler(func(c *core.Context) {
		nextCalled = true
	})

	mw := CORS(CORSOptions{AllowedOrigins: []string{"https://example.com"}})
	c.SetHandlers([]core.Handler{mw, h})
	c.Next()

	if !nextCalled {
		t.Fatal("handler should be called")
	}
	if got := string(fctx.Response.Header.Peek("Access-Control-Allow-Origin")); got != "" {
		t.Fatalf("expected no Allow-Origin header, got %q", got)
	}
}

// ===== Rate Limit =====

func TestRateLimit_ExceedsLimit(t *testing.T) {
	mw := RateLimit(2, time.Hour)

	for i := 0; i < 2; i++ {
		var fctx fasthttp.RequestCtx
		c := core.NewContext(&fctx, nil, nil)
		called := false
		h := core.Handler(func(c *core.Context) { called = true })
		c.SetHandlers([]core.Handler{mw, h})
		c.Next()
		if !called {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	var fctx fasthttp.RequestCtx
	c := core.NewContext(&fctx, nil, nil)
	called := false
	h := core.Handler(func(c *core.Context) { called = true })
	c.SetHandlers([]core.Handler{mw, h})
	c.Next()

	if called {
		t.Fatal("request should be blocked after limit")
	}
	if fctx.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", fctx.Response.StatusCode())
	}
	if !c.Aborted() {
		t.Fatal("expected context to be aborted")
	}
	if got := string(fctx.Response.Header.Peek("Retry-After")); got == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestRateLimit_TrustForwarded(t *testing.T) {
	mw := RateLimit(1, time.Hour, RateLimitOptions{TrustForwarded: true})

	blocked := func() bool {
		var fctx fasthttp.RequestCtx
		fctx.Request.Header.Set("X-Forwarded-For", "1.2.3.4")
		c := core.NewContext(&fctx, nil, nil)
		called := false
		h := core.Handler(func(c *core.Context) { called = true })
		c.SetHandlers([]core.Handler{mw, h})
		c.Next()
		return called
	}

	if !blocked() {
		t.Fatal("first request should pass")
	}
	if blocked() {
		t.Fatal("second request from same IP should be blocked")
	}

	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("X-Forwarded-For", "5.6.7.8")
	c := core.NewContext(&fctx, nil, nil)
	called := false
	h := core.Handler(func(c *core.Context) { called = true })
	c.SetHandlers([]core.Handler{mw, h})
	c.Next()
	if !called {
		t.Fatal("different IP should not be blocked")
	}
}

// ===== Compress =====

func TestCompress_Gzip(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("Accept-Encoding", "gzip")
	c := core.NewContext(&fctx, nil, nil)

	payload := strings.Repeat("TurboGo compress middleware. ", 20)
	h := core.Handler(func(c *core.Context) {
		c.Text(200, payload)
	})

	c.SetHandlers([]core.Handler{Compress(9), h})
	c.Next()

	if got := string(fctx.Response.Header.Peek("Content-Encoding")); got != "gzip" {
		t.Fatalf("expected gzip encoding, got %q", got)
	}
	if got := string(fctx.Response.Header.Peek("Vary")); got != "Accept-Encoding" {
		t.Fatalf("expected Vary header, got %q", got)
	}

	body := fctx.Response.Body()
	if len(body) >= len(payload) {
		t.Fatalf("expected compressed body smaller than original")
	}

	gr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	decompressed, err := io.ReadAll(gr)
	if err != nil {
		t.Fatal(err)
	}
	if string(decompressed) != payload {
		t.Fatalf("decompressed body mismatch")
	}
}

func TestCompress_SkipSmallBody(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("Accept-Encoding", "gzip")
	c := core.NewContext(&fctx, nil, nil)

	h := core.Handler(func(c *core.Context) {
		c.Text(200, "tiny")
	})

	c.SetHandlers([]core.Handler{Compress(9), h})
	c.Next()

	if got := string(fctx.Response.Header.Peek("Content-Encoding")); got != "" {
		t.Fatalf("small body should not be compressed, got encoding %q", got)
	}
	if string(fctx.Response.Body()) != "tiny" {
		t.Fatalf("unexpected body %q", fctx.Response.Body())
	}
}

func TestCompress_NoAcceptEncoding(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := core.NewContext(&fctx, nil, nil)

	payload := strings.Repeat("plain body. ", 20)
	h := core.Handler(func(c *core.Context) {
		c.Text(200, payload)
	})

	c.SetHandlers([]core.Handler{Compress(9), h})
	c.Next()

	if got := string(fctx.Response.Header.Peek("Content-Encoding")); got != "" {
		t.Fatalf("expected no compression, got %q", got)
	}
}

// ===== Request ID =====

func TestRequestID_Generated(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := core.NewContext(&fctx, nil, nil)

	h := core.Handler(func(c *core.Context) {
		id, _ := c.GetSession("request_id").(string)
		if len(id) != 32 {
			t.Fatalf("expected 32-char id, got %q", id)
		}
	})

	c.SetHandlers([]core.Handler{RequestID(), h})
	c.Next()

	if got := string(fctx.Response.Header.Peek("X-Request-ID")); len(got) != 32 {
		t.Fatalf("expected response header id, got %q", got)
	}
}

func TestRequestID_FromClient(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("X-Request-ID", "client-provided-id")
	c := core.NewContext(&fctx, nil, nil)

	c.SetHandlers([]core.Handler{RequestID()})
	c.Next()

	if got := string(fctx.Response.Header.Peek("X-Request-ID")); got != "client-provided-id" {
		t.Fatalf("expected echoed id, got %q", got)
	}
}

func TestRequestID_CustomGenerator(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := core.NewContext(&fctx, nil, nil)

	mw := RequestID(RequestIDConfig{Generator: func() string { return "fixed-id" }})
	c.SetHandlers([]core.Handler{mw})
	c.Next()

	if got := string(fctx.Response.Header.Peek("X-Request-ID")); got != "fixed-id" {
		t.Fatalf("expected custom id, got %q", got)
	}
}

// ===== Timeout =====

func TestTimeout_CompletedInTime(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := core.NewContext(&fctx, nil, nil)

	called := false
	h := core.Handler(func(c *core.Context) {
		called = true
		c.Text(200, "ok")
	})

	c.SetHandlers([]core.Handler{Timeout(100 * time.Millisecond), h})
	c.Next()

	if !called {
		t.Fatal("handler should be called")
	}
	if fctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d", fctx.Response.StatusCode())
	}
}

func TestTimeout_Expired(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := core.NewContext(&fctx, nil, nil)

	h := core.Handler(func(c *core.Context) {
		select {}
	})
	c.SetHandlers([]core.Handler{h})

	Timeout(30 * time.Millisecond)(c)

	if fctx.Response.StatusCode() != fasthttp.StatusRequestTimeout {
		t.Fatalf("expected 408, got %d", fctx.Response.StatusCode())
	}
	if string(fctx.Response.Body()) != "request timeout" {
		t.Fatalf("unexpected body: %q", fctx.Response.Body())
	}
}

// ===== Basic Auth =====

func TestBasicAuth_Valid(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("Authorization", "Basic "+basicAuthEncode("admin", "secret"))
	c := core.NewContext(&fctx, nil, nil)

	called := false
	h := core.Handler(func(c *core.Context) {
		called = true
		if u, _ := c.GetSession("username").(string); u != "admin" {
			t.Fatalf("expected username admin, got %q", u)
		}
	})

	mw := BasicAuth([]BasicAuthUser{{Username: "admin", Password: "secret"}})
	c.SetHandlers([]core.Handler{mw, h})
	c.Next()

	if !called {
		t.Fatal("handler should be called")
	}
}

func TestBasicAuth_Invalid(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("Authorization", "Basic "+basicAuthEncode("admin", "wrong"))
	c := core.NewContext(&fctx, nil, nil)

	called := false
	h := core.Handler(func(c *core.Context) { called = true })

	mw := BasicAuth([]BasicAuthUser{{Username: "admin", Password: "secret"}})
	c.SetHandlers([]core.Handler{mw, h})
	c.Next()

	if called {
		t.Fatal("handler should not be called")
	}
	if fctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", fctx.Response.StatusCode())
	}
	if got := string(fctx.Response.Header.Peek("WWW-Authenticate")); !strings.Contains(got, "Basic") {
		t.Fatalf("expected WWW-Authenticate header, got %q", got)
	}
}

func TestBasicAuth_MissingHeader(t *testing.T) {
	var fctx fasthttp.RequestCtx
	c := core.NewContext(&fctx, nil, nil)

	mw := BasicAuth([]BasicAuthUser{{Username: "admin", Password: "secret"}})
	c.SetHandlers([]core.Handler{mw})
	c.Next()

	if fctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", fctx.Response.StatusCode())
	}
	if !c.Aborted() {
		t.Fatal("expected context to be aborted")
	}
}

func TestBasicAuth_WithValidator(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.Set("Authorization", "Basic "+basicAuthEncode("budi", "pwd123"))
	c := core.NewContext(&fctx, nil, nil)

	called := false
	h := core.Handler(func(c *core.Context) { called = true })

	mw := BasicAuthWithValidator(func(username, password string) bool {
		return username == "budi" && password == "pwd123"
	})
	c.SetHandlers([]core.Handler{mw, h})
	c.Next()

	if !called {
		t.Fatal("handler should be called")
	}
}

// ===== CSRF =====

func TestCSRF_SafeMethodSetsCookie(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetMethod("GET")
	c := core.NewContext(&fctx, nil, nil)

	called := false
	h := core.Handler(func(c *core.Context) { called = true })

	c.SetHandlers([]core.Handler{CSRF(), h})
	c.Next()

	if !called {
		t.Fatal("handler should be called")
	}
	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)
	cookie.SetKey("csrf_token")
	if !fctx.Response.Header.Cookie(cookie) {
		t.Fatal("expected csrf cookie in response")
	}
	if len(cookie.Value()) == 0 {
		t.Fatal("expected non-empty csrf token")
	}
}

func TestCSRF_UnsafeMethodWithoutToken(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetMethod("POST")
	c := core.NewContext(&fctx, nil, nil)

	called := false
	h := core.Handler(func(c *core.Context) { called = true })

	c.SetHandlers([]core.Handler{CSRF(), h})
	c.Next()

	if called {
		t.Fatal("handler should not be called")
	}
	if fctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("expected 403, got %d", fctx.Response.StatusCode())
	}
}

func TestCSRF_ValidToken(t *testing.T) {
	token := "valid-token-value"

	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetMethod("POST")
	fctx.Request.Header.SetCookie("csrf_token", token)
	fctx.Request.Header.Set("X-CSRF-Token", token)
	c := core.NewContext(&fctx, nil, nil)

	called := false
	h := core.Handler(func(c *core.Context) { called = true })

	c.SetHandlers([]core.Handler{CSRF(), h})
	c.Next()

	if !called {
		t.Fatal("handler should be called with valid token")
	}
}

func TestCSRF_MismatchedToken(t *testing.T) {
	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetMethod("POST")
	fctx.Request.Header.SetCookie("csrf_token", "cookie-token")
	fctx.Request.Header.Set("X-CSRF-Token", "header-token")
	c := core.NewContext(&fctx, nil, nil)

	called := false
	h := core.Handler(func(c *core.Context) { called = true })

	c.SetHandlers([]core.Handler{CSRF(), h})
	c.Next()

	if called {
		t.Fatal("handler should not be called")
	}
	if fctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("expected 403, got %d", fctx.Response.StatusCode())
	}
}

// ===== Logger =====

func TestLogger_WritesRequestLine(t *testing.T) {
	var buf bytes.Buffer

	var fctx fasthttp.RequestCtx
	fctx.Request.Header.SetMethod("GET")
	fctx.Request.SetRequestURI("/hello")
	c := core.NewContext(&fctx, nil, nil)

	h := core.Handler(func(c *core.Context) {
		c.Text(200, "ok")
	})

	mw := Logger(LoggerConfig{Output: &buf, DisableColors: true})
	c.SetHandlers([]core.Handler{mw, h})
	c.Next()

	out := buf.String()
	if !strings.Contains(out, "GET") || !strings.Contains(out, "/hello") || !strings.Contains(out, "200") {
		t.Fatalf("unexpected log output: %q", out)
	}
}