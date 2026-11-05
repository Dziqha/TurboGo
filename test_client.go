package TurboGo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/valyala/fasthttp"
)

// Test runs req through the full application (router, middleware, recovery)
// without opening a network listener, and returns the recorded response.
// It is meant for tests:
//
//	req := httptest.NewRequest(http.MethodGet, "/user/42", nil)
//	resp := app.Test(req)
//	if resp.StatusCode != http.StatusOK { ... }
//	body, _ := io.ReadAll(resp.Body)
//
// The response body is buffered, so it stays readable after the call.
// The handler is cached after the first call; routes registered later are
// still matched because the router rebuilds its table lazily.
func (a *App) Test(req *http.Request) *http.Response {
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&fasthttp.Request{}, nil, nil)

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(req.URL.RequestURI())
	switch {
	case req.Host != "":
		ctx.Request.Header.SetHost(req.Host)
	case req.URL.Host != "":
		ctx.Request.Header.SetHost(req.URL.Host)
	}
	for k, vs := range req.Header {
		switch http.CanonicalHeaderKey(k) {
		case "Host", "Content-Length", "Transfer-Encoding":
			continue // set by fasthttp from SetHost/SetBody
		}
		for _, v := range vs {
			ctx.Request.Header.Add(k, v)
		}
	}
	if req.Body != nil && req.Body != http.NoBody {
		if body, err := io.ReadAll(req.Body); err == nil {
			ctx.Request.SetBody(body)
		}
	}

	a.Handler()(ctx)

	code := ctx.Response.StatusCode()
	resp := &http.Response{
		Status:        fmt.Sprintf("%d %s", code, http.StatusText(code)),
		StatusCode:    code,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header, 8),
		Request:       req,
		ContentLength: int64(ctx.Response.Header.ContentLength()),
	}
	ctx.Response.Header.VisitAll(func(k, v []byte) {
		resp.Header.Add(string(k), string(v))
	})

	body := append([]byte(nil), ctx.Response.Body()...)
	if len(body) > 0 {
		resp.ContentLength = int64(len(body))
		resp.Body = io.NopCloser(bytes.NewReader(body))
	} else {
		resp.ContentLength = 0
		resp.Body = http.NoBody
	}
	return resp
}
