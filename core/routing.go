package core

import (
	"fmt"
	"time"

	"github.com/Dziqha/TurboGo/internal/concurrency"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

type Handler func(*Context)

type Router interface {
	Use(args ...any) Router

	Get(path string, handler Handler, handlers ...Handler) *Route
	Head(path string, handler Handler, handlers ...Handler) *Route
	Post(path string, handler Handler, handlers ...Handler) *Route
	Put(path string, handler Handler, handlers ...Handler) *Route
	Delete(path string, handler Handler, handlers ...Handler) *Route
	Connect(path string, handler Handler, handlers ...Handler) *Route
	Options(path string, handler Handler, handlers ...Handler) *Route
	Trace(path string, handler Handler, handlers ...Handler) *Route
	Patch(path string, handler Handler, handlers ...Handler) *Route

	Add(methods []string, path string, handler Handler, handlers ...Handler) *Route
	All(path string, handler Handler, handlers ...Handler) *Route

	Group(prefix string, handlers ...Handler) Router
	Route(path string) *Route
}

type routeOptions struct {
	ttl     *time.Duration
	disable bool
}

type Route struct {
	Method   string
	Path     string
	Name     string
	Handlers []Handler
	Options  routeOptions
	app      RouterApp
	methods  []string
}

type RouterApp interface {
	GetRoutes() []*Route
	SetRoutes([]*Route)
	GetMiddleware() []Handler
	GetRouter() *router.Router
	WrapHandlers([]Handler) fasthttp.RequestHandler
}

func (r *Route) NoCache() *Route {
	r.Options.disable = true
	return r
}

func (r *Route) Cache(ttl time.Duration) *Route {
	r.Options.ttl = &ttl
	r.Options.disable = false
	return r
}

func (r *Route) Named(name string) *Route {
	r.Name = name
	return r
}

func AddRoute(app RouterApp, methods []string, path string, handler Handler, handlers ...Handler) *Route {
	allHandlers := append([]Handler{handler}, handlers...)
	finalHandlers := append(app.GetMiddleware(), allHandlers...)

	route := &Route{
		Path:     path,
		Handlers: finalHandlers,
		app:      app,
		methods:  methods,
		Options: routeOptions{disable: false},
	}

	routes := app.GetRoutes()
	routes = append(routes, route)
	app.SetRoutes(routes)

	for _, method := range methods {
		route.Method = method

		ttl := 5 * time.Minute
		if route.Options.ttl != nil {
			ttl = *route.Options.ttl
		}

		var handlerChain []Handler
		if route.Options.disable {
			handlerChain = finalHandlers
		} else {
			handlerChain = withCacheInjection(method, path, finalHandlers, ttl)
		}

		app.GetRouter().Handle(method, path, app.WrapHandlers(handlerChain))
	}

	return route
}

func withCacheInjection(method, path string, handlers []Handler, ttl time.Duration) []Handler {
	cacheKey := fmt.Sprintf("cache:%s:%s", method, path)

	cacheMiddleware := func(c *Context) {
		start := time.Now()

		if val, ok := c.Cache.Memory.Get(cacheKey); ok {
			status := c.Ctx.Response.StatusCode()
			if status == 0 {
				status = fasthttp.StatusOK
			}

			c.Ctx.SetBody(val)
			c.Abort()

			ns := max(time.Since(start).Nanoseconds(), 100)
			Log.Debug("[CACHE] HIT    : %s %s [%3d] (%s)", method, path, status, formatDuration(ns))
			return
		}

		Log.Debug("[CACHE] MISS   : %s %s", method, path)

		original := c.Ctx.Response.Body()
		c.Ctx.Response.ResetBody()

		c.Next()

		status := c.Ctx.Response.StatusCode()
		if status == 0 {
			status = fasthttp.StatusOK
		}

		// Simpan ke cache secara async (non-blocking)
		bodyCopy := c.Ctx.Response.Body()
		concurrency.Async(func() {
			c.Cache.Memory.Set(cacheKey, bodyCopy, ttl)
			Log.Debug("[CACHE] STORED : %s (TTL: %v)", cacheKey, ttl)
		})

		ns := max(time.Since(start).Nanoseconds(), 100)
		Log.Debug("→ %-7s %-30s [%3d] (%s)", method, path, status, formatDuration(ns))

		c.Ctx.Response.AppendBody(original)
	}

	return append([]Handler{cacheMiddleware}, handlers...)
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func formatDuration(ns int64) string {
	switch {
	case ns >= 1e9:
		return fmt.Sprintf("%.3fs", float64(ns)/1e9)
	case ns >= 1e6:
		return fmt.Sprintf("%.3fms", float64(ns)/1e6)
	case ns >= 1e3:
		return fmt.Sprintf("%.3fµs", float64(ns)/1e3)
	default:
		return fmt.Sprintf("%dns", ns)
	}
}

// ==================== GROUP ====================

type Group struct {
	Prefix     string
	Parent     RouterApp
	Middleware []Handler
}

func (g *Group) Add(methods []string, path string, h Handler, hs ...Handler) *Route {
	fullPath := g.Prefix + path
	allHandlers := append([]Handler{h}, hs...)
	finalHandlers := append(g.Middleware, allHandlers...)
	return AddRoute(g.Parent, methods, fullPath, finalHandlers[0], finalHandlers[1:]...)
}

func (g *Group) Use(args ...any) Router {
	for _, arg := range args {
		if h, ok := arg.(Handler); ok {
			g.Middleware = append(g.Middleware, h)
		}
	}
	return g
}

func (g *Group) Get(path string, h Handler, hs ...Handler) *Route {
	return g.Add([]string{"GET"}, path, h, hs...)
}

func (g *Group) Post(path string, h Handler, hs ...Handler) *Route {
	return g.Add([]string{"POST"}, path, h, hs...)
}

func (g *Group) Put(path string, h Handler, hs ...Handler) *Route {
	return g.Add([]string{"PUT"}, path, h, hs...)
}

func (g *Group) Delete(path string, h Handler, hs ...Handler) *Route {
	return g.Add([]string{"DELETE"}, path, h, hs...)
}

func (g *Group) Head(path string, h Handler, hs ...Handler) *Route {
	return g.Add([]string{"HEAD"}, path, h, hs...)
}

func (g *Group) Patch(path string, h Handler, hs ...Handler) *Route {
	return g.Add([]string{"PATCH"}, path, h, hs...)
}

func (g *Group) Connect(path string, h Handler, hs ...Handler) *Route {
	return g.Add([]string{"CONNECT"}, path, h, hs...)
}

func (g *Group) Options(path string, h Handler, hs ...Handler) *Route {
	return g.Add([]string{"OPTIONS"}, path, h, hs...)
}

func (g *Group) Trace(path string, h Handler, hs ...Handler) *Route {
	return g.Add([]string{"TRACE"}, path, h, hs...)
}

func (g *Group) All(path string, h Handler, hs ...Handler) *Route {
	return g.Add([]string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}, path, h, hs...)
}

func (g *Group) Route(path string) *Route {
	for _, r := range g.Parent.GetRoutes() {
		if r.Path == path {
			return r
		}
	}
	return nil
}

func (g *Group) Group(prefix string, handlers ...Handler) Router {
	return &Group{
		Prefix:     g.Prefix + prefix,
		Parent:     g.Parent,
		Middleware: append(g.Middleware, handlers...),
	}
}
