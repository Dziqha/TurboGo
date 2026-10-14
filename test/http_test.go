package test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Dziqha/TurboGo"
	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

func doRequest(app *TurboGo.App, method, path string, fn func(req *fasthttp.Request)) *fasthttp.Response {
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&fasthttp.Request{}, nil, nil)

	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(path)
	if fn != nil {
		fn(&ctx.Request)
	}

	app.Handler()(ctx)

	ctx.Response.Body()
	resp := &fasthttp.Response{}
	ctx.Response.CopyTo(resp)
	return resp
}

func TestHandler_StaticFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("static content"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := TurboGo.New()
	app.Static("/static", dir)

	resp := doRequest(app, "GET", "/static/hello.txt", nil)
	if resp.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode())
	}
	if string(resp.Body()) != "static content" {
		t.Fatalf("unexpected body: %q", resp.Body())
	}

	notFound := doRequest(app, "GET", "/static/missing.txt", nil)
	if notFound.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("expected 404 for missing file, got %d", notFound.StatusCode())
	}
}

func TestHandler_Redirect(t *testing.T) {
	app := TurboGo.New()
	app.Get("/old", func(c *core.Context) {
		c.Redirect(fasthttp.StatusFound, "/new")
	})

	resp := doRequest(app, "GET", "/old", func(req *fasthttp.Request) {
		req.Header.SetHost("localhost:8080")
	})
	if resp.StatusCode() != fasthttp.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode())
	}
	if loc := string(resp.Header.Peek("Location")); loc != "http://localhost:8080/new" {
		t.Fatalf("expected Location http://localhost:8080/new, got %q", loc)
	}
}

func TestHandler_Cookies(t *testing.T) {
	app := TurboGo.New()
	app.Get("/set", func(c *core.Context) {
		cookie := fasthttp.AcquireCookie()
		cookie.SetKey("session")
		cookie.SetValue("abc123")
		cookie.SetPath("/")
		c.SetCookie(cookie)
		fasthttp.ReleaseCookie(cookie)
		c.Text(200, "ok")
	})
	app.Get("/read", func(c *core.Context) {
		c.Text(200, c.Cookie("session"))
	})
	app.Get("/delete", func(c *core.Context) {
		c.DeleteCookie("session")
		c.Text(200, "deleted")
	})

	setResp := doRequest(app, "GET", "/set", nil)
	setCookie := fasthttp.AcquireCookie()
	setCookie.SetKey("session")
	if !setResp.Header.Cookie(setCookie) {
		t.Fatal("expected session cookie in response")
	}
	if string(setCookie.Value()) != "abc123" {
		t.Fatalf("expected session cookie value abc123, got %q", setCookie.Value())
	}
	fasthttp.ReleaseCookie(setCookie)

	readResp := doRequest(app, "GET", "/read", func(req *fasthttp.Request) {
		req.Header.SetCookie("session", "abc123")
	})
	if string(readResp.Body()) != "abc123" {
		t.Fatalf("expected abc123, got %q", readResp.Body())
	}

	deleteResp := doRequest(app, "GET", "/delete", nil)
	deleted := fasthttp.AcquireCookie()
	deleted.SetKey("session")
	if !deleteResp.Header.Cookie(deleted) {
		t.Fatal("expected delete cookie in response")
	}
	if deleted.Expire().After(time.Now()) {
		t.Fatalf("expected expired cookie, got expiry %v", deleted.Expire())
	}
	fasthttp.ReleaseCookie(deleted)
}

type searchQuery struct {
	Page  int      `query:"page"`
	Limit int      `query:"limit"`
	Tags  []string `query:"tag"`
	Q     string   `query:"q"`
}

func TestHandler_BindQuery(t *testing.T) {
	app := TurboGo.New()
	app.Get("/search", func(c *core.Context) {
		var q searchQuery
		if err := c.BindQuery(&q); err != nil {
			c.JSON(400, map[string]any{"error": err.Error()})
			return
		}
		c.JSON(200, q)
	})

	resp := doRequest(app, "GET", "/search?page=2&limit=10&tag=a&tag=b&q=hello", nil)
	if resp.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), resp.Body())
	}

	var got searchQuery
	if err := json.Unmarshal(resp.Body(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Page != 2 || got.Limit != 10 || got.Q != "hello" {
		t.Fatalf("unexpected bind result: %+v", got)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "a" || got.Tags[1] != "b" {
		t.Fatalf("unexpected tags: %+v", got.Tags)
	}
}

type signupForm struct {
	Name  string `form:"name"`
	Email string `form:"email"`
	Age   int    `form:"age"`
	Admin bool   `form:"admin"`
}

func TestHandler_BindForm(t *testing.T) {
	app := TurboGo.New()
	app.Post("/signup", func(c *core.Context) {
		var f signupForm
		if err := c.BindForm(&f); err != nil {
			c.JSON(400, map[string]any{"error": err.Error()})
			return
		}
		c.JSON(200, f)
	})

	body := "name=budi&email=budi%40g.com&age=25&admin=true"
	resp := doRequest(app, "POST", "/signup", func(req *fasthttp.Request) {
		req.Header.SetContentType("application/x-www-form-urlencoded")
		req.SetBodyString(body)
	})
	if resp.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), resp.Body())
	}

	var got signupForm
	if err := json.Unmarshal(resp.Body(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "budi" || got.Email != "budi@g.com" || got.Age != 25 || !got.Admin {
		t.Fatalf("unexpected bind result: %+v", got)
	}
}

func TestHandler_FileUpload(t *testing.T) {
	app := TurboGo.New()
	app.Post("/upload", func(c *core.Context) {
		fh, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, map[string]any{"error": err.Error()})
			return
		}
		dst := filepath.Join(os.TempDir(), "turbogo-test-upload.txt")
		if err := c.SaveFile(fh, dst); err != nil {
			c.JSON(500, map[string]any{"error": err.Error()})
			return
		}
		defer os.Remove(dst)
		c.Text(200, fh.Filename)
	})

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("file content")); err != nil {
		t.Fatal(err)
	}
	mw.Close()

	resp := doRequest(app, "POST", "/upload", func(req *fasthttp.Request) {
		req.Header.SetContentType(mw.FormDataContentType())
		req.SetBody(buf.Bytes())
	})
	if resp.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode(), resp.Body())
	}
	if string(resp.Body()) != "hello.txt" {
		t.Fatalf("unexpected response: %q", resp.Body())
	}
}

func TestHandler_Templates(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("Hello {{.Name}}!"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "user.html"), []byte("User: {{.Name}}"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := TurboGo.New().Templates(dir)
	app.Get("/", func(c *core.Context) {
		c.HTML(200, "index.html", map[string]string{"Name": "TurboGo"})
	})
	app.Get("/user", func(c *core.Context) {
		c.HTML(200, "user.html", map[string]string{"Name": "Budi"})
	})

	resp := doRequest(app, "GET", "/", nil)
	if resp.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode())
	}
	if string(resp.Body()) != "Hello TurboGo!" {
		t.Fatalf("unexpected body: %q", resp.Body())
	}

	userResp := doRequest(app, "GET", "/user", nil)
	if string(userResp.Body()) != "User: Budi" {
		t.Fatalf("unexpected body: %q", userResp.Body())
	}
}

func TestHandler_CustomNotFound(t *testing.T) {
	app := TurboGo.New()
	app.NotFound(func(c *core.Context) {
		c.JSON(404, map[string]any{"error": "custom not found"})
	})

	resp := doRequest(app, "GET", "/does-not-exist", nil)
	if resp.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode())
	}
	if !strings.Contains(string(resp.Body()), "custom not found") {
		t.Fatalf("unexpected body: %s", resp.Body())
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	app := TurboGo.New()
	app.Post("/data", func(c *core.Context) {})

	resp := doRequest(app, "GET", "/data", nil)
	if resp.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode())
	}
}

func TestHandler_HeadFallbackToGet(t *testing.T) {
	app := TurboGo.New()
	app.Get("/ping", func(c *core.Context) {
		c.Text(200, "pong")
	})

	resp := doRequest(app, "HEAD", "/ping", nil)
	if resp.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200 for HEAD fallback, got %d", resp.StatusCode())
	}
	if len(resp.Body()) != 0 {
		t.Fatalf("HEAD should have empty body, got %q", resp.Body())
	}
}

func TestHandler_Params(t *testing.T) {
	app := TurboGo.New()
	app.Get("/user/:id", func(c *core.Context) {
		c.Text(200, "user-"+c.Param("id"))
	})

	resp := doRequest(app, "GET", "/user/42", nil)
	if resp.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode())
	}
	if string(resp.Body()) != "user-42" {
		t.Fatalf("expected user-42, got %q", resp.Body())
	}
}

func TestContext_SSE(t *testing.T) {
	app := TurboGo.New()
	app.Get("/sse", func(c *core.Context) {
		c.SSE(func(w *core.SSEWriter) {
			w.Event("ping", "hello")
			w.Data("world")
		})
	})

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&fasthttp.Request{}, nil, nil)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/sse")
	app.Handler()(ctx)

	if ct := string(ctx.Response.Header.ContentType()); ct != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", ct)
	}

	data, err := io.ReadAll(ctx.Response.BodyStream())
	if err != nil {
		t.Fatal(err)
	}
	want := "event: ping\ndata: hello\n\ndata: world\n\n"
	if string(data) != want {
		t.Fatalf("unexpected stream: %q", data)
	}
}
