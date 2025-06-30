package TurboGo

import (
	"strings"

	"github.com/Dziqha/TurboGo/core"
	"github.com/Dziqha/TurboGo/internal/cache"
	"github.com/Dziqha/TurboGo/internal/concurrency"
	"github.com/Dziqha/TurboGo/internal/pubsub"
	"github.com/Dziqha/TurboGo/internal/queue"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)


type App struct {
	routes     concurrency.SafeValue[[]*core.Route]
	middleware concurrency.SafeValue[[]core.Handler]
	router     *router.Router

	cache *cache.Engine 
	deps  *core.Dependencies 
}

const maxLineLength = 60

func Banner() string {
	return `
 ████████╗██╗   ██╗██████╗ ██████╗  ██████╗  ██████╗  ██████╗ 
 ╚══██╔══╝██║   ██║██╔══██╗██╔══██╗██╔═══██╗██╔════╝ ██╔═══██╗
    ██║   ██║   ██║██████╔╝██████╔╝██║   ██║██║  ███╗██║   ██║
    ██║   ██║   ██║██╔══██╗██╔══██╗██║   ██║██║   ██║██║   ██║
    ██║   ╚██████╔╝██║  ██║██████╔╝╚██████╔╝╚██████╔╝╚██████╔╝
    ╚═╝    ╚═════╝ ╚═╝  ╚═╝╚═════╝  ╚═════╝  ╚═════╝  ╚═════╝ 
                                                              
` + CenterText("High-Performance Web Framework for Go") + `
` + CenterText("Version: v1.0.0") + `                                                     
`
}

func CenterText(text string) string {
	textLen := len(text)
	padding := (maxLineLength - textLen) / 2
	return strings.Repeat(" ", padding) + text
}

func New() *App {
	cacheEngine, err := cache.NewEngine()
	if err != nil {
		panic("failed to initialize cache: " + err.Error())
	}

	pubsubEngine, err := pubsub.NewEngine()
	if err != nil {
		panic("failed to initialize pubsub: " + err.Error())
	}

	queueEngine, err := queue.NewEngine()
	if err != nil {
		panic("failed to initialize queue: " + err.Error())
	}

	deps := &core.Dependencies{
		Pubsub: pubsubEngine,
		Queue:  queueEngine,
	}

	return newApp(cacheEngine, deps)
}

func newApp(cache *cache.Engine, deps *core.Dependencies) *App {
	app := &App{
		router: router.New(),
		cache:  cache,
		deps:   deps,
	}
	app.routes.Set([]*core.Route{})
	app.middleware.Set([]core.Handler{})
	return app
}

func (a *App) Use(args ...any) core.Router {
	m := a.middleware.Get()
	for _, arg := range args {
		if h, ok := arg.(core.Handler); ok {
			m = append(m, h)
		}
	}
	a.middleware.Set(m)
	return a
}

func (a *App) Group(prefix string, handlers ...core.Handler) core.Router {
	return &core.Group{
		Prefix:     prefix,
		Parent:     a,
		Middleware: handlers,
	}
}

func (a *App) Route(path string) *core.Route {
	for _, r := range a.routes.Get() {
		if r.Path == path {
			return r
		}
	}
	return nil
}

func (a *App) wrap(handlers []core.Handler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		c := core.NewContext(ctx, a.cache, a.deps.Pubsub, a.deps.Queue, handlers)
		c.Next()
	}
}

func (a *App) RoutesInfo() []map[string]string {
	var result []map[string]string
	for _, r := range a.routes.Get() {
		result = append(result, map[string]string{
			"method": r.Method,
			"path":   r.Path,
			"name":   r.Name,
		})
	}
	return result
}

func (a *App) Listen(addr string) error {
	println(Banner())
	return fasthttp.ListenAndServe(addr, a.router.Handler)
}

// HTTP Methods
func (a *App) Get(path string, h core.Handler, hs ...core.Handler) *core.Route {
	return a.Add([]string{"GET"}, path, h, hs...)
}
func (a *App) Post(path string, h core.Handler, hs ...core.Handler) *core.Route {
	return a.Add([]string{"POST"}, path, h, hs...)
}
func (a *App) Put(path string, h core.Handler, hs ...core.Handler) *core.Route {
	return a.Add([]string{"PUT"}, path, h, hs...)
}
func (a *App) Delete(path string, h core.Handler, hs ...core.Handler) *core.Route {
	return a.Add([]string{"DELETE"}, path, h, hs...)
}
func (a *App) Head(path string, h core.Handler, hs ...core.Handler) *core.Route {
	return a.Add([]string{"HEAD"}, path, h, hs...)
}
func (a *App) Patch(path string, h core.Handler, hs ...core.Handler) *core.Route {
	return a.Add([]string{"PATCH"}, path, h, hs...)
}
func (a *App) Connect(path string, h core.Handler, hs ...core.Handler) *core.Route {
	return a.Add([]string{"CONNECT"}, path, h, hs...)
}
func (a *App) Options(path string, h core.Handler, hs ...core.Handler) *core.Route {
	return a.Add([]string{"OPTIONS"}, path, h, hs...)
}
func (a *App) Trace(path string, h core.Handler, hs ...core.Handler) *core.Route {
	return a.Add([]string{"TRACE"}, path, h, hs...)
}
func (a *App) All(path string, h core.Handler, hs ...core.Handler) *core.Route {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	return a.Add(methods, path, h, hs...)
}

func (a *App) Add(methods []string, path string, handler core.Handler, handlers ...core.Handler) *core.Route {
	return core.AddRoute(a, methods, path, handler, handlers...)
}

func (a *App) GetRoutes() []*core.Route {
	return a.routes.Get()
}
func (a *App) SetRoutes(routes []*core.Route) {
	a.routes.Set(routes)
}
func (a *App) GetMiddleware() []core.Handler {
	return a.middleware.Get()
}
func (a *App) GetRouter() *router.Router {
	return a.router
}
func (a *App) WrapHandlers(handlers []core.Handler) fasthttp.RequestHandler {
	return a.wrap(handlers)
}
