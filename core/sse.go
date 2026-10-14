package core

import (
	"bufio"
	"fmt"

	"github.com/valyala/fasthttp"
)

type SSEWriter struct {
	w    *bufio.Writer
	done <-chan struct{}
}

func (c *Context) SSE(fn func(w *SSEWriter)) {
	ctx := c.Ctx
	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		sw := &SSEWriter{w: w, done: safeDone(ctx)}
		fn(sw)
	})
}

func safeDone(ctx *fasthttp.RequestCtx) <-chan struct{} {
	defer func() { _ = recover() }()
	return ctx.Done()
}

func (w *SSEWriter) write(s string) {
	w.w.WriteString(s)
	w.w.Flush()
}

func (w *SSEWriter) Event(event, data string) {
	w.write("event: " + event + "\n")
	w.write("data: " + data + "\n\n")
}

func (w *SSEWriter) Data(data string) {
	w.write("data: " + data + "\n\n")
}

func (w *SSEWriter) Comment(text string) {
	w.write(":" + text + "\n\n")
}

func (w *SSEWriter) Retry(ms int) {
	w.write(fmt.Sprintf("retry: %d\n\n", ms))
}

func (w *SSEWriter) Done() <-chan struct{} {
	return w.done
}

func (w *SSEWriter) Flush() {
	w.w.Flush()
}