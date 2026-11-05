package TurboGo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/Dziqha/TurboGo/core"
)

func testReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if resp.ContentLength != int64(len(b)) && resp.ContentLength != -1 {
		t.Fatalf("ContentLength=%d but body is %d bytes", resp.ContentLength, len(b))
	}
	return string(b)
}

func TestAppTest_GetWithMiddlewareAndHeaders(t *testing.T) {
	app := New()
	app.Use(func(c *core.Context) {
		c.Ctx.Response.Header.Set("X-Middleware", "ran")
	})
	app.Get("/hello", func(c *core.Context) {
		c.Ctx.Response.Header.Set("X-Echo", c.Header("X-Token"))
		c.Text(201, "hi")
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req.Header.Set("X-Token", "secret")
	resp := app.Test(req)

	if resp.StatusCode != 201 {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Middleware"); got != "ran" {
		t.Fatalf("middleware header = %q", got)
	}
	if got := resp.Header.Get("X-Echo"); got != "secret" {
		t.Fatalf("echo header = %q, want secret", got)
	}
	if body := testReadBody(t, resp); body != "hi" {
		t.Fatalf("body = %q, want hi", body)
	}
	if resp.Request != req {
		t.Fatal("resp.Request not set to the request")
	}
	if resp.Status != "201 Created" {
		t.Fatalf("status line = %q", resp.Status)
	}
}

func TestAppTest_PathParamAndQuery(t *testing.T) {
	app := New()
	app.Get("/user/:id", func(c *core.Context) {
		c.Text(200, c.Param("id")+":"+c.Query("tab"))
	})

	resp := app.Test(httptest.NewRequest(http.MethodGet, "/user/42?tab=posts", nil))
	if got := testReadBody(t, resp); got != "42:posts" {
		t.Fatalf("body = %q, want 42:posts", got)
	}
}

func TestAppTest_JSONPostBody(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	app := New()
	app.Post("/echo", func(c *core.Context) {
		var in payload
		if err := c.BindJSON(&in); err != nil {
			c.Text(400, err.Error())
			return
		}
		c.JSON(200, payload{Name: strings.ToUpper(in.Name)})
	})

	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"name":"ada"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := app.Test(req)

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, testReadBody(t, resp))
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type = %q", ct)
	}
	var out payload
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Name != "ADA" {
		t.Fatalf("name = %q, want ADA", out.Name)
	}
}

func TestAppTest_NotFoundAndMethodNotAllowed(t *testing.T) {
	app := New()
	app.Get("/only-get", func(c *core.Context) { c.Text(200, "ok") })

	if resp := app.Test(httptest.NewRequest(http.MethodGet, "/missing", nil)); resp.StatusCode != 404 {
		t.Fatalf("missing route status = %d, want 404", resp.StatusCode)
	}
	if resp := app.Test(httptest.NewRequest(http.MethodPost, "/only-get", nil)); resp.StatusCode != 405 {
		t.Fatalf("wrong method status = %d, want 405", resp.StatusCode)
	}
}

func TestAppTest_PanicBecomes500(t *testing.T) {
	app := New()
	app.Get("/panic", func(c *core.Context) { panic("boom") })

	resp := app.Test(httptest.NewRequest(http.MethodGet, "/panic", nil))
	if resp.StatusCode != 500 {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

func TestAppTest_EmptyBody(t *testing.T) {
	app := New()
	app.Get("/empty", func(c *core.Context) { c.Status(204) })

	resp := app.Test(httptest.NewRequest(http.MethodGet, "/empty", nil))
	if resp.Body != http.NoBody {
		t.Fatal("empty response should expose http.NoBody")
	}
	if resp.ContentLength != 0 {
		t.Fatalf("ContentLength = %d, want 0", resp.ContentLength)
	}
}

// Routes added after the first Test call must still match: the cached
// handler relies on the router rebuilding its table lazily.
func TestAppTest_LateRoutesStillMatch(t *testing.T) {
	app := New()
	app.Get("/first", func(c *core.Context) { c.Text(200, "first") })
	if resp := app.Test(httptest.NewRequest(http.MethodGet, "/first", nil)); resp.StatusCode != 200 {
		t.Fatalf("first status = %d", resp.StatusCode)
	}

	app.Get("/second", func(c *core.Context) { c.Text(200, "second") })
	resp := app.Test(httptest.NewRequest(http.MethodGet, "/second", nil))
	if resp.StatusCode != 200 || testReadBody(t, resp) != "second" {
		t.Fatalf("late route: status = %d", resp.StatusCode)
	}
}

func TestAppTest_Concurrent(t *testing.T) {
	app := New()
	app.Get("/user/:id", func(c *core.Context) { c.Text(200, c.Param("id")) })

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				resp := app.Test(httptest.NewRequest(http.MethodGet, "/user/7", nil))
				if resp.StatusCode != 200 {
					t.Errorf("status = %d", resp.StatusCode)
					return
				}
				if body := testReadBody(t, resp); body != "7" {
					t.Errorf("body = %q", body)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestAppTest_NoGoroutineLeak(t *testing.T) {
	app := New()
	app.Get("/ping", func(c *core.Context) { c.Text(200, "pong") })

	app.Test(httptest.NewRequest(http.MethodGet, "/ping", nil)) // warm up
	runtime.GC()
	before := runtime.NumGoroutine()

	for i := 0; i < 5000; i++ {
		resp := app.Test(httptest.NewRequest(http.MethodGet, "/ping", nil))
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	}
	runtime.GC()
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Fatalf("goroutines grew: %d -> %d", before, after)
	}
}
