package test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Dziqha/TurboGo"
	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details any    `json:"details"`
		Fields  any    `json:"fields"`
	} `json:"error"`
}

func decodeError(t *testing.T, resp *fasthttp.Response) errorBody {
	t.Helper()
	var body errorBody
	if err := json.Unmarshal(resp.Body(), &body); err != nil {
		t.Fatalf("invalid error json %q: %v", resp.Body(), err)
	}
	return body
}

func TestError_HTTPError(t *testing.T) {
	app := TurboGo.New()
	app.Get("/e", func(c *core.Context) {
		c.Error(core.NewHTTPError(fasthttp.StatusNotFound, "USER_NOT_FOUND", "user not found").
			WithDetails(map[string]string{"id": "42"}))
	})

	resp := doRequest(app, "GET", "/e", nil)
	if resp.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode())
	}
	body := decodeError(t, resp)
	if body.Error.Code != "USER_NOT_FOUND" || body.Error.Message != "user not found" {
		t.Fatalf("unexpected error body: %+v", body)
	}
	if body.Error.Details == nil {
		t.Fatal("expected details in error body")
	}
}

func TestError_ValidationError(t *testing.T) {
	app := TurboGo.New()
	app.Post("/e", func(c *core.Context) {
		type payload struct {
			Name  string `json:"name" validate:"required"`
			Email string `json:"email" validate:"email"`
		}
		var p payload
		if err := c.Bind(&p); err != nil {
			c.Error(err)
			return
		}
		c.Text(fasthttp.StatusOK, "ok")
	})

	resp := doRequest(app, "POST", "/e", func(req *fasthttp.Request) {
		req.Header.SetContentType("application/json")
		req.SetBodyString(`{"name":"","email":"bad"}`)
	})
	if resp.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode())
	}
	body := decodeError(t, resp)
	if body.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", body.Error.Code)
	}
	fields, ok := body.Error.Fields.([]any)
	if !ok || len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %v", body.Error.Fields)
	}
}

func TestError_PlainErrorIs500(t *testing.T) {
	app := TurboGo.New()
	app.Get("/e", func(c *core.Context) {
		c.Error(errors.New("database exploded"))
	})

	resp := doRequest(app, "GET", "/e", nil)
	if resp.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode())
	}
	body := decodeError(t, resp)
	if body.Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("expected INTERNAL_ERROR, got %q", body.Error.Code)
	}
	if body.Error.Message == "database exploded" {
		t.Fatal("internal error message must not leak to client")
	}
}

func TestError_ErrorStopsChain(t *testing.T) {
	app := TurboGo.New()
	app.Use(func(c *core.Context) {
		c.Error(core.ErrUnauthorized)
	})
	app.Get("/e", func(c *core.Context) {
		c.Text(fasthttp.StatusOK, "should not reach")
	})

	resp := doRequest(app, "GET", "/e", nil)
	if resp.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode())
	}
}

func TestError_PanicIsRecovered(t *testing.T) {
	app := TurboGo.New()
	app.Get("/panic", func(c *core.Context) {
		panic("boom")
	})

	resp := doRequest(app, "GET", "/panic", nil)
	if resp.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode())
	}
	body := decodeError(t, resp)
	if body.Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("expected INTERNAL_ERROR, got %q", body.Error.Code)
	}
}

func TestError_PanicWithHTTPError(t *testing.T) {
	app := TurboGo.New()
	app.Get("/panic", func(c *core.Context) {
		panic(core.ErrNotFound)
	})

	resp := doRequest(app, "GET", "/panic", nil)
	if resp.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode())
	}
}

func TestError_CustomErrorHandler(t *testing.T) {
	app := TurboGo.New()
	app.SetErrorHandler(func(c *core.Context, err error) {
		c.Text(fasthttp.StatusTeapot, "custom: "+err.Error())
	})
	app.Get("/panic", func(c *core.Context) {
		panic(errors.New("kaboom"))
	})

	resp := doRequest(app, "GET", "/panic", nil)
	if resp.StatusCode() != fasthttp.StatusTeapot {
		t.Fatalf("expected 418, got %d", resp.StatusCode())
	}
	if string(resp.Body()) != "custom: kaboom" {
		t.Fatalf("unexpected body: %q", resp.Body())
	}
}

func TestUse_AcceptsFuncLiteral(t *testing.T) {
	app := TurboGo.New()
	app.Use(func(c *core.Context) {
		c.SetSession("mw", true)
	})
	app.Get("/e", func(c *core.Context) {
		if c.GetSession("mw") == true {
			c.Text(fasthttp.StatusAccepted, "mw ran")
		} else {
			c.Text(fasthttp.StatusOK, "no mw")
		}
	})

	resp := doRequest(app, "GET", "/e", nil)
	if resp.StatusCode() != fasthttp.StatusAccepted {
		t.Fatalf("expected 202 from middleware, got %d", resp.StatusCode())
	}
}

func TestError_NextRequestUnaffected(t *testing.T) {
	app := TurboGo.New()
	app.Get("/panic", func(c *core.Context) {
		panic("boom")
	})
	app.Get("/ok", func(c *core.Context) {
		c.Text(fasthttp.StatusOK, "fine")
	})

	first := doRequest(app, "GET", "/panic", nil)
	if first.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", first.StatusCode())
	}

	second := doRequest(app, "GET", "/ok", nil)
	if second.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d", second.StatusCode())
	}
	if string(second.Body()) != "fine" {
		t.Fatalf("unexpected body: %q", second.Body())
	}
}
