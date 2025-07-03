package core

import (
	"encoding/json"

	"github.com/Dziqha/TurboGo/internal/cache"
	"github.com/Dziqha/TurboGo/internal/concurrency"
	"github.com/Dziqha/TurboGo/internal/pubsub"
	"github.com/Dziqha/TurboGo/internal/queue"

	"github.com/valyala/fasthttp"
)

type Context struct {
	Ctx   *fasthttp.RequestCtx
	Cache *cache.Engine
	Pubsub *pubsub.Engine
	Queue *queue.Engine
	index    int
	handlers []Handler
	aborted bool
	values map[string]any

}

type Dependencies struct {
	Pubsub *pubsub.Engine
	Queue  *queue.Engine
}


func NewContext(ctx *fasthttp.RequestCtx, cache *cache.Engine, pubsub *pubsub.Engine, queue *queue.Engine, handlers []Handler) *Context {
	return &Context{
		Ctx:      ctx,
		Cache:    cache,
		Pubsub:    pubsub,
		Queue:    queue,
		handlers: handlers,
		index:    -1, // ⬅️  agar Next() mulai dari 0
	}
}


// JSON writes a JSON response
func (c *Context) JSON(status int, data any) {
	c.Ctx.SetStatusCode(status)
	c.Ctx.SetContentType("application/json")

	res, err := json.Marshal(data)
	if err != nil {
		c.Ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		c.Ctx.SetBodyString(`{"error":"failed to marshal json"}`)
		return
	}
	c.Ctx.SetBody(res)
}



// Text writes a plain text response
func (c *Context) Text(code int, msg string) {
	c.Ctx.SetStatusCode(code)
	c.Ctx.SetContentType("text/plain")
	c.Ctx.SetBodyString(msg)
}

// Param returns a URL parameter (from user-defined routing system)
func (c *Context) Param(key string) string {
	// implementasi tergantung routing kamu
	val := c.Ctx.UserValue(key)
	if val == nil {
		return ""
	}
	return val.(string)
}

// Query returns a query parameter
func (c *Context) Query(key string) string {
	return string(c.Ctx.QueryArgs().Peek(key))
}

// Body binds JSON body to a struct
func (c *Context) BindJSON(dest any) error {
	return json.Unmarshal(c.Ctx.PostBody(), dest)
}

func (c *Context) Header(key string) string {
    return string(c.Ctx.Request.Header.Peek(key))
}

func (c *Context) Next() {
	c.index++
	if c.index < len(c.handlers) && !c.aborted {
		c.handlers[c.index](c)
	}
}

func (c *Context) Abort() {
	c.aborted = true
}

func (c *Context) Aborted() bool {
	return c.aborted
}


// Async menjalankan fungsi dalam goroutine tanpa block
func (c *Context) Async(fn func()) {
	concurrency.Async(fn)
}

// Parallel menjalankan banyak fungsi lalu menunggu semuanya selesai
func (c *Context) Parallel(funcs ...func()) {
	concurrency.WaitGroupRunner(funcs...)
}


func (c *Context) SetSession(key string, value any) {
	if c.values == nil {
		c.values = make(map[string]any)
	}
	c.values[key] = value
}

func (c *Context) GetSession(key string) any {
	if c.values == nil {
		return nil
	}
	if val, ok := c.values[key]; ok {
		if str, ok := val.(string);  ok {
			return  str
		}
	}

	return  ""
}