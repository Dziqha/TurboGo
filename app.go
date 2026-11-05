package TurboGo

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/Dziqha/TurboGo/core"
	"github.com/Dziqha/TurboGo/internal/cache"
	"github.com/Dziqha/TurboGo/internal/pubsub"
	"github.com/Dziqha/TurboGo/internal/queue"
	"github.com/Dziqha/TurboGo/internal/router"
	"github.com/valyala/fasthttp"
)

func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

const maxLineLength = 50

func CenterText(text string) string {
	textLen := len(text)
	if textLen >= maxLineLength {
		return text
	}
	padding := (maxLineLength - textLen) / 2
	return strings.Repeat(" ", padding) + text
}

func Banner(addr string) string {
	return `
 _____            _             ___      
/__   \_   _ _ __| |__   ___   / _ \___  
  / /\/ | | | '__| '_ \ / _ \ / /_\/ _ \ 
 / /  | |_| | |  | |_) | (_) / /_\\ (_) |
 \/    \__,_|_|  |_.__/ \___/\____/\___/ 
                                         
` +
		CenterText("🌀 TurboGo Ultra High-Performance Web Framework") + "\n" +
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
	templates    *template.Template
	srv          *fasthttp.Server
	ErrorHandler func(*core.Context, error)
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
		switch h := arg.(type) {
		case core.Handler:
			a.middleware = append(a.middleware, h)
		case func(*core.Context):
			a.middleware = append(a.middleware, core.Handler(h))
		}
	}
	return a
}

func (a *App) Route(path string) *router.Route {
	for _, r := range a.routes {
		if r.Path == path {
			return r
		}
	}
	return nil
}

func (a *App) newServer() *fasthttp.Server {
	// Handler() freezes routing into the compiled-switch table (see there).
	srv := &fasthttp.Server{
		Handler:                       a.Handler(),
		Logger:                        log.New(io.Discard, "", 0),
		LogAllErrors:                  false,
		ReadBufferSize:                4096,
		WriteBufferSize:               4096,
		Concurrency:                   256 * 1024,
		MaxConnsPerIP:                 10000,
		MaxRequestsPerConn:            0,
		DisableKeepalive:              false,
		TCPKeepalive:                  false,
		ReduceMemoryUsage:             false,
		NoDefaultServerHeader:         true,
		NoDefaultDate:                 true,
		NoDefaultContentType:          true,
		DisableHeaderNamesNormalizing: true,
		DisablePreParseMultipartForm:  true,
		StreamRequestBody:             false,
		GetOnly:                       false,
	}
	a.srv = srv
	return srv
}

func (a *App) RunServer(addr string) error {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println(Banner(addr))
	return a.newServer().ListenAndServe(addr)
}

func (a *App) RunTLSServer(addr, certFile, keyFile string) error {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println(Banner(addr))
	return a.newServer().ListenAndServeTLS(addr, certFile, keyFile)
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.srv == nil {
		return errors.New("server is not running")
	}
	return a.srv.ShutdownWithContext(ctx)
}

func (a *App) RunWithGracefulShutdown(addr string) error {
	errCh := make(chan error, 1)
	go func() {
		if err := a.RunServer(addr); err != nil {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-errCh:
		return err
	case <-quit:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return a.Shutdown(shutdownCtx)
}

func (a *App) Handler() fasthttp.RequestHandler {
	// Freeze routing like newServer does, so mounting Handler() under a
	// custom fasthttp server gets the compiled table too. Late routes
	// still mark it dirty and rebuild lazily.
	a.singleRouter.SetCompiled(true)
	return func(ctx *fasthttp.RequestCtx) {
		method := b2s(ctx.Method())
		path, _ := routePath(ctx)

		c := core.NewContext(ctx, a.cache, nil)
		c.Templates = a.templates
		defer core.ReleaseContext(c)

		defer func() {
			if rec := recover(); rec != nil {
				if err, ok := rec.(error); ok {
					a.handleError(c, err)
				} else {
					a.handleError(c, fmt.Errorf("%v", rec))
				}
			}
		}()
		handler, route, nparams := a.singleRouter.FindInto(method, path, c.Pslots(), c.Params())
		c.SetNparams(nparams)

		if route != nil {
			if route.FastHandler != nil {
				core.ReleaseContext(c)
				route.FastHandler(ctx)
				return
			}
			hs := route.Handlers
			if a.pubsub != nil || a.queue != nil {
				if a.pubsub != nil {
					c.SetPubsub(a.pubsub)
				}
				if a.queue != nil {
					c.SetQueue(a.queue)
				}
			}
			if len(hs) == 1 {
				hs[0](c)
			} else {
				c.SetHandlers(hs)
				c.Next()
			}
		} else {
			c.SetHandlers([]core.Handler{handler})
			c.Next()
		}

		if method == "HEAD" {
			ctx.Response.ResetBody()
		}
	}
}

func (a *App) handleError(c *core.Context, err error) {
	if a.ErrorHandler != nil {
		a.ErrorHandler(c, err)
		return
	}
	c.Error(err)
}

func (a *App) SetErrorHandler(fn func(*core.Context, error)) *App {
	a.ErrorHandler = fn
	return a
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

	handlerChain := make([]core.Handler, len(a.middleware)+len(handlers))
	copy(handlerChain, a.middleware)
	copy(handlerChain[len(a.middleware):], handlers)

	route := &router.Route{
		Path:     path,
		Method:   methods[0],
		Handlers: handlerChain,
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
					if err, ok := rec.(error); ok {
						a.handleError(c, err)
					} else {
						a.handleError(c, fmt.Errorf("%v", rec))
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

func (a *App) Static(prefix, dir string) *App {
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	prefix = strings.TrimSuffix(prefix, "/")

	fs := &fasthttp.FS{
		Root:               dir,
		IndexNames:         []string{"index.html"},
		GenerateIndexPages: false,
		Compress:           true,
		CompressBrotli:     true,
		CacheDuration:      10 * time.Second,
		PathRewrite: func(ctx *fasthttp.RequestCtx) []byte {
			p := ctx.Path()
			if len(p) >= len(prefix) {
				p = p[len(prefix):]
			}
			if len(p) == 0 {
				p = []byte("/")
			}
			return p
		},
	}
	handler := fs.NewRequestHandler()

	a.singleRouter.AddStaticFast("GET", prefix, handler)
	a.singleRouter.AddFastRoute("GET", prefix+"/", handler)
	a.singleRouter.AddFastRoute("GET", prefix+"/*", handler)
	return a
}

func (a *App) Templates(dir string) *App {
	if a.templates == nil {
		a.templates = template.New("")
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		name, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = a.templates.New(filepath.ToSlash(name)).Parse(string(data))
		return err
	})
	if err != nil {
		panic("failed to load templates: " + err.Error())
	}
	return a
}

func (a *App) NotFound(h core.Handler) *App {
	a.singleRouter.SetNotFoundHandler(h)
	return a
}

func (a *App) MethodNotAllowed(h core.Handler) *App {
	a.singleRouter.SetMethodNotAllowedHandler(h)
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
		c.Templates = a.templates
		defer core.ReleaseContext(c)
		defer func() {
			if rec := recover(); rec != nil {
				if err, ok := rec.(error); ok {
					a.handleError(c, err)
				} else {
					a.handleError(c, fmt.Errorf("%v", rec))
				}
			}
		}()
		c.Next()
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

func (a *App) GetFast(path string, handler fasthttp.RequestHandler) *router.Route {
	return a.singleRouter.AddStaticFast("GET", path, handler)
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
