package core

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/Dziqha/TurboGo/internal/cache"
	"github.com/Dziqha/TurboGo/internal/concurrency"
	"github.com/Dziqha/TurboGo/internal/pubsub"
	"github.com/Dziqha/TurboGo/internal/queue"

	"github.com/valyala/fasthttp"
)

type Handler func(*Context)
type Context struct {
	Ctx      *fasthttp.RequestCtx
	Cache    *cache.Engine
	Pubsub   *pubsub.Engine
	Queue    *queue.Engine
	index    int
	handlers []Handler
	aborted  bool
	values   map[string]any // untuk menyimpan session atau data lainnya
	Writer   *strings.Builder
	params   map[string]string // untuk route parameters
}

type EngineContext struct {
	Pubsub *pubsub.Engine
	Queue  *queue.Engine
}

var contextPool = sync.Pool{
	New: func() any {
		builder := &strings.Builder{}
		builder.Grow(512) // Pre-allocate 512 bytes capacity
		return &Context{
			Writer: builder,
			params: make(map[string]string),
			values: make(map[string]any),
		}
	},
}

func NewContext(ctx *fasthttp.RequestCtx, cache *cache.Engine, handlers []Handler) *Context {
	c := contextPool.Get().(*Context)

	// Reset semua field
	c.Ctx = ctx
	c.Cache = cache
	c.Pubsub = nil
	c.Queue = nil
	c.index = -1 // agar Next() mulai dari index 0
	c.handlers = handlers
	c.aborted = false

	if c.params == nil {
		c.params = make(map[string]string)
	} else {
		for k := range c.params {
			delete(c.params, k)
		}
	}

	if c.values == nil {
		c.values = make(map[string]any)
	} else {
		for k := range c.values {
			delete(c.values, k)
		}
	}

	if c.Writer == nil {
		builder := &strings.Builder{}
		builder.Grow(512) // Pre-allocate 512 bytes capacity
		c.Writer = builder
	} else {
		c.Writer.Reset() // Reset writer untuk reuse
	}

	return c
}

func ReleaseContext(c *Context) {
	c.Ctx = nil
	c.Cache = nil
	c.Pubsub = nil
	c.Queue = nil
	c.handlers = nil
	c.aborted = false
	c.index = -1

	for k := range c.params {
		delete(c.params, k)
	}
	for k := range c.values {
		delete(c.values, k)
	}

	contextPool.Put(c)
}

func NewEngineContext() *EngineContext {
	return &EngineContext{}
}

func (c *Context) Next() {
	c.index++
	for c.index < len(c.handlers) {
		if c.aborted {
			break
		}
		c.handlers[c.index](c)
		c.index++
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

func (c *Context) Text(code int, msg string) {
	c.Ctx.SetStatusCode(code)
	c.Ctx.SetContentType("text/plain")
	c.Ctx.SetBodyString(msg)
}

func (c *Context) Param(key string) string {
	if val, exists := c.params[key]; exists {
		return val
	}

	val := c.Ctx.UserValue(key)
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
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

func (c *Context) Abort() {
	c.aborted = true
}

func (c *Context) Aborted() bool {
	return c.aborted
}

func (c *Context) Async(fn func()) {
	concurrency.Async(fn)
}

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
		return val
	}
	return nil
}

func (c *Context) MustPubsub() *pubsub.Engine {
	if c.Pubsub == nil {
		panic("🚨 Pubsub engine is not set in Context. Use SetPubsub() before calling this.")
	}
	return c.Pubsub
}

func (c *Context) MustQueue() *queue.Engine {
	if c.Queue == nil {
		panic("🚨 Queue engine is not set in Context. Use SetQueue() before calling this.")
	}
	return c.Queue
}

func (c *Context) SetQueue(q *queue.Engine) {
	c.Queue = q
}

func (c *Context) SetPubsub(p *pubsub.Engine) {
	c.Pubsub = p
}

func (c *Context) SetParam(key, value string) {
	if c.params == nil {
		c.params = make(map[string]string)
	}
	c.params[key] = value

	if c.Ctx != nil {
		c.Ctx.SetUserValue(key, value)
	}
}

func (c *Context) Status(code int) *Context {
	c.Ctx.SetStatusCode(code)
	return c
}

func (c *Context) SendString(s string) *Context {
	c.Ctx.SetBodyString(s)
	return c
}

func (c *Context) GetAllParams() map[string]string {
	result := make(map[string]string)
	for k, v := range c.params {
		result[k] = v
	}
	return result
}
