package TurboGo

import (
	"github.com/Dziqha/TurboGo/core"
	"github.com/Dziqha/TurboGo/internal/kafka"
	"github.com/Dziqha/TurboGo/internal/rabbitmq"
	"github.com/Dziqha/TurboGo/internal/redis"
	"strings"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

type App struct {
	routes     []*core.Route
	middleware []core.Handler
	router     *router.Router

	redis *redis.Engine
	kafka *kafka.Engine
	queue *rabbitmq.Engine
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

// New adalah factory function untuk membuat instance App baru
func New() *App {
    // Inisialisasi internal engines
    redisEngine, err := redis.NewEngine()
    if err != nil {
        panic("failed to initialize Redis: " + err.Error())
    }

    kafkaEngine, err := kafka.NewEngine()
    if err != nil {
        panic("failed to initialize Kafka: " + err.Error())
    }

    queueEngine, err := rabbitmq.NewEngine()
    if err != nil {
        panic("failed to initialize RabbitMQ: " + err.Error())
    }

    return newApp(redisEngine, kafkaEngine, queueEngine)
}


func newApp(redis *redis.Engine, kafka *kafka.Engine, queue *rabbitmq.Engine) *App {
	return &App{
		routes: []*core.Route{},
		router: router.New(),
		redis:  redis,
		kafka:  kafka,
		queue:  queue,
	}
}

func (a *App) Use(args ...any) core.Router {
	for _, arg := range args {
		if h, ok := arg.(core.Handler); ok {
			a.middleware = append(a.middleware, h)
		}
	}
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
	for _, r := range a.routes {
		if r.Path == path {
			return r
		}
	}
	return nil
}

func (a *App) wrap(handlers []core.Handler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		c := core.NewContext(ctx, a.redis, a.kafka, a.queue, handlers)
		c.Next()
	}
}

func (a *App) RoutesInfo() []map[string]string {
	var result []map[string]string
	for _, r := range a.routes {
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

// HTTP Method implementations
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

// Implement RouterApp interface
func (a *App) GetRoutes() []*core.Route {
	return a.routes
}

func (a *App) SetRoutes(routes []*core.Route) {
	a.routes = routes
}

func (a *App) GetMiddleware() []core.Handler {
	return a.middleware
}

func (a *App) GetRouter() *router.Router {
	return a.router
}

func (a *App) WrapHandlers(handlers []core.Handler) fasthttp.RequestHandler {
	return a.wrap(handlers)
}