package core

import (
	"encoding/json"
	"html/template"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Dziqha/TurboGo/internal/cache"
	"github.com/Dziqha/TurboGo/internal/concurrency"
	"github.com/Dziqha/TurboGo/internal/pubsub"
	"github.com/Dziqha/TurboGo/internal/queue"

	"github.com/valyala/fasthttp"
)

type Handler func(*Context)

// ParamSlot is one named path parameter. Route params live here first
// (written directly by the router, zero hashing); the params map is only
// materialized when map access is actually needed. See Params().
type ParamSlot struct {
	K string
	V string
}

// MaxSlots bounds inline param storage (see ParamSlot). Routes with more
// params spill to the params map via the overflow path (correct, just
// slower). Exposed for the router's zero-alloc lookup.
const MaxSlots = 8

type Context struct {
	Ctx       *fasthttp.RequestCtx
	Cache     *cache.Engine
	Pubsub    *pubsub.Engine
	Queue     *queue.Engine
	Templates *template.Template
	index     atomic.Int32
	handlers  []Handler
	aborted   atomic.Bool
	values    map[string]any
	Writer    *strings.Builder
	params    map[string]string
	pslots    [MaxSlots]ParamSlot
	nparams   int
}

type EngineContext struct {
	Pubsub *pubsub.Engine
	Queue  *queue.Engine
}

var contextPool = sync.Pool{
	New: func() any {
		builder := &strings.Builder{}
		builder.Grow(512)
		return &Context{
			Writer: builder,
			params: make(map[string]string),
			values: make(map[string]any),
		}
	},
}

func NewContext(ctx *fasthttp.RequestCtx, cache *cache.Engine, handlers []Handler) *Context {
	c := contextPool.Get().(*Context)

	c.Ctx = ctx
	c.Cache = cache
	c.Pubsub = nil
	c.Queue = nil
	c.Templates = nil
	c.index.Store(-1)
	c.handlers = handlers
	c.aborted.Store(false)
	c.nparams = 0

	clear(c.params)
	clear(c.values)

	if c.Writer == nil {
		builder := &strings.Builder{}
		builder.Grow(512)
		c.Writer = builder
	} else {
		c.Writer.Reset()
	}

	return c
}

func ReleaseContext(c *Context) {
	c.Ctx = nil
	c.Cache = nil
	c.Pubsub = nil
	c.Queue = nil
	c.Templates = nil
	c.handlers = nil
	c.aborted.Store(false)
	c.index.Store(-1)

	contextPool.Put(c)
}

func NewEngineContext() *EngineContext {
	return &EngineContext{}
}

func (c *Context) Params() map[string]string {
	// Materialize slots into the map (map wins on conflict: a handler
	// SetParam override must survive). Idempotent.
	for i := 0; i < c.nparams; i++ {
		k := c.pslots[i].K
		if _, ok := c.params[k]; !ok {
			c.params[k] = c.pslots[i].V
		}
	}
	return c.params
}

// Pslots exposes the inline param slots for zero-alloc router writes.
// See SingleThreadedRouter.FindInto.
func (c *Context) Pslots() *[MaxSlots]ParamSlot {
	return &c.pslots
}

// SetNparams records how many slots FindInto filled.
func (c *Context) SetNparams(n int) {
	c.nparams = n
}

func (c *Context) SetHandlers(h []Handler) {
	c.handlers = h
	c.index.Store(-1)
}

func (c *Context) Next() {
	c.index.Add(1)
	n := len(c.handlers)
	if n == 0 {
		return
	}
	if n == 1 && c.index.Load() == 0 && !c.aborted.Load() {
		c.handlers[0](c)
		c.index.Add(1)
		return
	}
	for c.index.Load() < int32(n) {
		if c.aborted.Load() {
			break
		}
		c.handlers[c.index.Load()](c)
		c.index.Add(1)
	}
}

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
	for i := 0; i < c.nparams; i++ {
		if c.pslots[i].K == key {
			return c.pslots[i].V
		}
	}
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

func (c *Context) Query(key string) string {
	return string(c.Ctx.QueryArgs().Peek(key))
}

func (c *Context) BindJSON(dest any) error {
	return json.Unmarshal(c.Ctx.PostBody(), dest)
}

func (c *Context) Header(key string) string {
	return string(c.Ctx.Request.Header.Peek(key))
}

func (c *Context) Abort() {
	c.aborted.Store(true)
}

func (c *Context) Aborted() bool {
	return c.aborted.Load()
}

func (c *Context) Async(fn func()) {
	concurrency.Async(fn)
}

func (c *Context) Parallel(funcs ...func()) {
	concurrency.WaitGroupRunner(funcs...)
}

func (c *Context) SetSession(key string, value any) {
	if c.values == nil {
		c.values = make(map[string]any, 2)
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
	for i := 0; i < c.nparams; i++ {
		if c.pslots[i].K == key {
			c.pslots[i].V = value
		}
	}
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
	result := make(map[string]string, len(c.params)+c.nparams)
	for k, v := range c.params {
		result[k] = v
	}
	for i := 0; i < c.nparams; i++ {
		if _, ok := result[c.pslots[i].K]; !ok {
			result[c.pslots[i].K] = c.pslots[i].V
		}
	}
	return result
}

func (c *Context) SetParams(p map[string]string) {
	clear(c.params)
	for k, v := range p {
		c.params[k] = v
		if c.Ctx != nil {
			c.Ctx.SetUserValue(k, v)
		}
	}
	// Full replace: slots are dead.
	c.nparams = 0
}

func (c *Context) Redirect(status int, url string) {
	c.Ctx.Redirect(url, status)
}

func (c *Context) SetCookie(cookie *fasthttp.Cookie) {
	c.Ctx.Response.Header.SetCookie(cookie)
}

func (c *Context) Cookie(name string) string {
	return string(c.Ctx.Request.Header.Cookie(name))
}

func (c *Context) DeleteCookie(name string) {
	cookie := fasthttp.AcquireCookie()
	cookie.SetKey(name)
	cookie.SetValue("")
	cookie.SetPath("/")
	cookie.SetExpire(time.Unix(0, 0))
	c.Ctx.Response.Header.SetCookie(cookie)
	fasthttp.ReleaseCookie(cookie)
}

func (c *Context) FormFile(key string) (*multipart.FileHeader, error) {
	return c.Ctx.FormFile(key)
}

func (c *Context) SaveFile(fh *multipart.FileHeader, dst string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

func (c *Context) MultipartForm() (*multipart.Form, error) {
	return c.Ctx.MultipartForm()
}

func (c *Context) HTML(status int, name string, data any) {
	c.Ctx.SetStatusCode(status)
	c.Ctx.SetContentType("text/html; charset=utf-8")

	if c.Templates == nil {
		c.Ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		c.Ctx.SetBodyString("templates are not loaded")
		return
	}

	if err := c.Templates.ExecuteTemplate(c.Ctx.Response.BodyWriter(), name, data); err != nil {
		c.Ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		c.Ctx.SetBodyString("failed to render template: " + err.Error())
	}
}
