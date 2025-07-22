package TurboGo

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/Dziqha/TurboGo/core"
	"github.com/Dziqha/TurboGo/internal/cache"
	"github.com/Dziqha/TurboGo/internal/pubsub"
	"github.com/Dziqha/TurboGo/internal/queue"
	"github.com/Dziqha/TurboGo/internal/router"
	"github.com/fatih/color"
	"github.com/valyala/fasthttp"
)

const maxLineLength = 60

func CenterText(text string) string {
	textLen := len(text)
	padding := (maxLineLength - textLen) / 2
	return strings.Repeat(" ", padding) + text
}

func Banner(addr string) string {
	return `
 ████████╗██╗   ██╗██████╗ ██████╗  ██████╗  ██████╗  ██████╗ 
 ╚══██╔══╝██║   ██║██╔══██╗██╔══██╗██╔═══██╗██╔════╝ ██╔═══██╗
    ██║   ██║   ██║██████╔╝██████╔╝██║   ██║██║  ███╗██║   ██║
    ██║   ██║   ██║██╔══██╗██╔══██╗██║   ██║██║   ██║██║   ██║
    ██║   ╚██████╔╝██║  ██║██████╔╝╚██████╔╝╚██████╔╝╚██████╔╝
    ╚═╝    ╚═════╝ ╚═╝  ╚═╝╚═════╝  ╚═════╝  ╚═════╝  ╚═════╝ 
` +
		CenterText("🌀 TurboGo Ultra High-Performance Web Framework") + "\n" +
		CenterText("Version: v3.0.0 - Memory Optimized Edition") + "\n" +
		CenterText(fmt.Sprintf("⚡ Listening on: http://localhost%s", addr)) + "\n"
}

type App struct {
	routes     []*router.Route
	middleware []core.Handler

	singleRouter *router.SingleThreadedRouter
	EngineCtx    *core.EngineContext
	cache        *cache.Engine
	pubsub       *pubsub.Engine
	queue        *queue.Engine
}

type Group struct {
	prefix     string
	middleware []core.Handler
	app        *App
}

func New() *App {
	return &App{
		routes:       make([]*router.Route, 0, 8),
		middleware:   make([]core.Handler, 0, 2),
		singleRouter: router.NewSingleThreadedRouter(),
		EngineCtx:    core.NewEngineContext(),
	}
}

func (a *App) WithCache() *App {
	if a.cache == nil {
		cacheEngine, err := cache.NewEngine()
		if err != nil {
			panic("failed to initialize cache: " + err.Error())
		}
		a.cache = cacheEngine
	}
	return a
}

func (a *App) Use(args ...any) *App {
	for _, arg := range args {
		if h, ok := arg.(core.Handler); ok {
			a.middleware = append(a.middleware, h)
		}
	}
	return a
}

func loggerWrap(c *core.Context, _ []core.Handler) {
	if core.DisableLogger {
		return
	}

	status := c.Ctx.Response.StatusCode()
	if status == 0 {
		if c.Aborted() {
			status = fasthttp.StatusUnauthorized
		} else {
			status = fasthttp.StatusOK
		}
		c.Ctx.SetStatusCode(status)
	}

	method := string(c.Ctx.Method())
	path := string(c.Ctx.Path())
	timestamp := time.Now().Format("15:04:05")

	var statusColor *color.Color

	switch {
	case status >= 200 && status < 300:
		statusColor = color.New(color.FgGreen)
	case status >= 300 && status < 400:
		statusColor = color.New(color.FgCyan)
	case status >= 400 && status < 500:
		statusColor = color.New(color.FgYellow)
	default:
		statusColor = color.New(color.FgRed)
	}

	methodColor := color.New(color.FgMagenta).SprintFunc()
	timeColor := color.New(color.FgBlue).SprintFunc()
	statusStr := statusColor.Sprintf("%d", status)

	fmt.Printf("🌀 TurboGo [%s] %s %s [%s]\n",
		timeColor(timestamp),
		methodColor(method),
		path,
		statusStr,
	)

}

func loggerMiddleware(c *core.Context) {
	loggerWrap(c, nil)
}

func (a *App) Route(path string) *router.Route {
	for _, r := range a.routes {
		if r.Path == path {
			return r
		}
	}
	return nil
}

func (a *App) RunServer(addr string) error {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println(Banner(addr))
	fmt.Printf("🔥 Using %d CPU cores\n", runtime.NumCPU())
	return fasthttp.ListenAndServe(addr, a.Handler())
}

func (a *App) Handler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		method := string(ctx.Method())
		path := string(ctx.Path())
		handler, route, params := a.singleRouter.Find(method, path)

		allHandlers := make([]core.Handler, 0, len(a.middleware)+1)
		allHandlers = append(allHandlers, a.middleware...)
		allHandlers = append(allHandlers, handler)
		allHandlers = append(allHandlers, loggerMiddleware)

		c := core.NewContext(ctx, a.cache, allHandlers)

		if len(params) > 0 {
			for k, v := range params {
				c.SetParam(k, v)
			}
		}

		if a.pubsub != nil {
			c.SetPubsub(a.pubsub)
		}
		if a.queue != nil {
			c.SetQueue(a.queue)
		}

		if route != nil {
			a.routes = append(a.routes, route)
		}

		c.Next()
		defer core.ReleaseContext(c)
	}
}

func (a *App) Add(methods []string, path string, h core.Handler, hs ...core.Handler) *router.Route {
	if len(methods) == 0 {
		panic("methods cannot be empty")
	}
	if h == nil {
		panic("primary handler cannot be nil")
	}

	handlers := make([]core.Handler, 0, len(hs)+1)
	handlers = append(handlers, h)
	handlers = append(handlers, hs...)

	route := &router.Route{
		Path:     path,
		Method:   methods[0],
		Handlers: append(a.middleware, handlers...),
		Options:  router.RouteOptions{Disable: true},
	}
	a.routes = append(a.routes, route)

	for _, method := range methods {
		finalHandlers := make([]core.Handler, 0, len(a.middleware)+len(handlers))
		finalHandlers = append(finalHandlers, a.middleware...)
		finalHandlers = append(finalHandlers, handlers...)

		finalHandler := func(c *core.Context) {
			defer func() {
				if rec := recover(); rec != nil {
					fmt.Printf("Panic in handler: %v\n", rec)
					if c != nil && c.Ctx != nil {
						c.Ctx.SetStatusCode(500)
						c.Ctx.SetBodyString("Internal Server Error")
					}
				}
			}()

			for _, handler := range finalHandlers {
				handler(c)
				if c.Aborted() {
					break
				}
			}

		}

		if strings.Contains(path, ":") {
			a.singleRouter.AddParametric(method, path, finalHandler, route)
		} else if strings.Contains(path, "*") {
			prefix := strings.TrimSuffix(path, "*")
			a.singleRouter.AddWildcard(method, prefix, finalHandler, route)
		} else {
			a.singleRouter.AddStatic(method, path, finalHandler, route)
		}
	}

	return route
}

func (a *App) WithQueue() *App {
	if a.queue == nil {
		qe, err := queue.NewEngine()
		if err != nil {
			panic("failed to initialize queue: " + err.Error())
		}
		a.EngineCtx.Queue = qe
		a.queue = qe
	}
	return a
}

func (a *App) WithPubsub() *App {
	if a.pubsub == nil {
		ps, err := pubsub.NewEngine()
		if err != nil {
			panic("failed to initialize pubsub: " + err.Error())
		}
		a.EngineCtx.Pubsub = ps
		a.pubsub = ps
	}
	return a
}

func (a *App) WrapHandlers(hs []core.Handler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		c := core.NewContext(ctx, a.cache, hs)
		c.Next()
		defer core.ReleaseContext(c)
	}
}

func (a *App) Group(prefix string, middlewares ...core.Handler) *Group {
	return &Group{
		prefix:     prefix,
		middleware: middlewares,
		app:        a,
	}
}

func (g *Group) Add(methods []string, path string, h core.Handler, hs ...core.Handler) *router.Route {
	fullPath := g.prefix + path
	allHandlers := make([]core.Handler, 0, len(g.middleware)+len(hs)+1)
	allHandlers = append(allHandlers, g.middleware...)
	allHandlers = append(allHandlers, h)
	allHandlers = append(allHandlers, hs...)

	return g.app.Add(methods, fullPath, allHandlers[0], allHandlers[1:]...)
}

// HTTP method shortcuts
func (a *App) Get(path string, handler core.Handler, handlers ...core.Handler) *router.Route {
	return a.Add([]string{"GET"}, path, handler, handlers...)
}

func (a *App) Post(path string, handler core.Handler, handlers ...core.Handler) *router.Route {
	return a.Add([]string{"POST"}, path, handler, handlers...)
}

func (a *App) Put(path string, handler core.Handler, handlers ...core.Handler) *router.Route {
	return a.Add([]string{"PUT"}, path, handler, handlers...)
}

func (a *App) Delete(path string, handler core.Handler, handlers ...core.Handler) *router.Route {
	return a.Add([]string{"DELETE"}, path, handler, handlers...)
}

func (a *App) Patch(path string, handler core.Handler, handlers ...core.Handler) *router.Route {
	return a.Add([]string{"PATCH"}, path, handler, handlers...)
}

func (a *App) Options(path string, handler core.Handler, handlers ...core.Handler) *router.Route {
	return a.Add([]string{"OPTIONS"}, path, handler, handlers...)
}

func (a *App) Head(path string, handler core.Handler, handlers ...core.Handler) *router.Route {
	return a.Add([]string{"HEAD"}, path, handler, handlers...)
}

func (a *App) All(path string, handler core.Handler, handlers ...core.Handler) *router.Route {
	return a.Add([]string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}, path, handler, handlers...)
}
func (a *App) Connect(path string, h core.Handler, hs ...core.Handler) *router.Route {
	return a.Add([]string{"CONNECT"}, path, h, hs...)
}
func (g *Group) Get(path string, h core.Handler, hs ...core.Handler) *router.Route {
	return g.Add([]string{"GET"}, path, h, hs...)
}

func (g *Group) Post(path string, h core.Handler, hs ...core.Handler) *router.Route {
	return g.Add([]string{"POST"}, path, h, hs...)
}

func (g *Group) Put(path string, h core.Handler, hs ...core.Handler) *router.Route {
	return g.Add([]string{"PUT"}, path, h, hs...)
}

func (g *Group) Delete(path string, h core.Handler, hs ...core.Handler) *router.Route {
	return g.Add([]string{"DELETE"}, path, h, hs...)
}

func (g *Group) Patch(path string, h core.Handler, hs ...core.Handler) *router.Route {
	return g.Add([]string{"PATCH"}, path, h, hs...)
}

func (g *Group) Options(path string, h core.Handler, hs ...core.Handler) *router.Route {
	return g.Add([]string{"OPTIONS"}, path, h, hs...)
}

func (g *Group) All(path string, h core.Handler, hs ...core.Handler) *router.Route {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	return g.Add(methods, path, h, hs...)
}
