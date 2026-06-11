package router

import (
	"strings"
	"time"

	"github.com/Dziqha/TurboGo/core"
)

type Route struct {
	Method   string
	Path     string
	Name     string
	Handlers []core.Handler
	Options  RouteOptions
}

type RouteOptions struct {
	Ttl     *time.Duration
	Disable bool
	Force   bool
}

type RouterConfig struct {
	MaxCacheSize       int64
	EvictionPolicy     string
	EnableMetrics      bool
	EnableCaching      bool
	EnablePooling      bool
	EnableUnsafeOpt    bool
	StaticPreAlloc     int
	WildcardPreAlloc   int
	ParametricPreAlloc int
}

type FastRoute struct {
	handler core.Handler
	route   *Route
}

type SingleThreadedRouter struct {
	staticGET    map[string]*FastRoute
	staticPOST   map[string]*FastRoute
	staticPUT    map[string]*FastRoute
	staticDELETE map[string]*FastRoute
	staticOther  map[string]*FastRoute

	tree *radixTree

	notFoundHandler         core.Handler
	methodNotAllowedHandler core.Handler
	config                  RouterConfig
}

func NewSingleThreadedRouter() *SingleThreadedRouter {
	r := &SingleThreadedRouter{
		staticGET:    make(map[string]*FastRoute, 8),
		staticPOST:   make(map[string]*FastRoute, 4),
		staticPUT:    make(map[string]*FastRoute, 2),
		staticDELETE: make(map[string]*FastRoute, 2),
		staticOther:  make(map[string]*FastRoute, 2),

		tree: newRadixTree(),

		notFoundHandler: func(c *core.Context) {
			c.Status(404).SendString("404 Not Found")
		},
		methodNotAllowedHandler: func(c *core.Context) {
			c.Status(405).SendString("405 Method Not Allowed")
		},
		config: DefaultRouterConfig(),
	}
	return r
}

func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		MaxCacheSize:       1000,
		EvictionPolicy:     "lru",
		EnableMetrics:      false,
		EnableCaching:      false,
		EnablePooling:      false,
		EnableUnsafeOpt:    true,
		StaticPreAlloc:     64,
		WildcardPreAlloc:   8,
		ParametricPreAlloc: 16,
	}
}

func (r *SingleThreadedRouter) AddStatic(method, path string, handler core.Handler, route *Route) {
	switch method {
	case "GET":
		r.staticGET[path] = &FastRoute{handler: handler, route: route}
	case "POST":
		r.staticPOST[path] = &FastRoute{handler: handler, route: route}
	case "PUT":
		r.staticPUT[path] = &FastRoute{handler: handler, route: route}
	case "DELETE":
		r.staticDELETE[path] = &FastRoute{handler: handler, route: route}
	default:
		r.staticOther[method+":"+path] = &FastRoute{handler: handler, route: route}
	}
	r.tree.insert(method, path, handler, route)
}

func (r *SingleThreadedRouter) AddWildcard(method, prefix string, handler core.Handler, route *Route) {
	r.tree.insert(method, prefix+"*", handler, route)
}

func (r *SingleThreadedRouter) AddParametric(method, pattern string, handler core.Handler, route *Route) {
	r.tree.insert(method, pattern, handler, route)
}

func (r *SingleThreadedRouter) Find(method, path string, params map[string]string) (core.Handler, *Route) {
	switch method {
	case "GET":
		if fr := r.staticGET[path]; fr != nil {
			return fr.handler, fr.route
		}
	case "POST":
		if fr := r.staticPOST[path]; fr != nil {
			return fr.handler, fr.route
		}
	case "PUT":
		if fr := r.staticPUT[path]; fr != nil {
			return fr.handler, fr.route
		}
	case "DELETE":
		if fr := r.staticDELETE[path]; fr != nil {
			return fr.handler, fr.route
		}
	default:
		if fr := r.staticOther[method+":"+path]; fr != nil {
			return fr.handler, fr.route
		}
	}

	handler, route, ok := r.tree.search(method, path, params)
	if ok {
		return handler, route
	}

	if r.tree.pathExistsAny(path) {
		return r.methodNotAllowedHandler, nil
	}

	return r.notFoundHandler, nil
}

func (r *SingleThreadedRouter) Reset() {
	r.staticGET = make(map[string]*FastRoute, 8)
	r.staticPOST = make(map[string]*FastRoute, 4)
	r.staticPUT = make(map[string]*FastRoute, 2)
	r.staticDELETE = make(map[string]*FastRoute, 2)
	r.staticOther = make(map[string]*FastRoute, 2)
	r.tree = newRadixTree()
}

func (r *SingleThreadedRouter) GetRouteCount() (static int) {
	static = len(r.staticGET) + len(r.staticPOST) + len(r.staticPUT) + len(r.staticDELETE) + len(r.staticOther)
	return
}

type RouterGroup struct {
	prefix string
	router *SingleThreadedRouter
}

func (r *SingleThreadedRouter) Group(prefix string) *RouterGroup {
	return &RouterGroup{prefix: prefix, router: r}
}

func (g *RouterGroup) add(methods []string, path string, handler core.Handler, route *Route) {
	fullPath := g.prefix + path
	for _, method := range methods {
		if strings.Contains(path, ":") {
			g.router.AddParametric(method, fullPath, handler, route)
		} else if strings.Contains(path, "*") {
			prefix := strings.TrimSuffix(fullPath, "*")
			g.router.AddWildcard(method, prefix, handler, route)
		} else {
			g.router.AddStatic(method, fullPath, handler, route)
		}
	}
}

func (g *RouterGroup) Get(path string, handler core.Handler) *Route {
	route := &Route{Method: "GET", Path: g.prefix + path}
	g.add([]string{"GET"}, path, handler, route)
	return route
}

func (g *RouterGroup) Post(path string, handler core.Handler) *Route {
	route := &Route{Method: "POST", Path: g.prefix + path}
	g.add([]string{"POST"}, path, handler, route)
	return route
}

func (g *RouterGroup) Put(path string, handler core.Handler) *Route {
	route := &Route{Method: "PUT", Path: g.prefix + path}
	g.add([]string{"PUT"}, path, handler, route)
	return route
}

func (g *RouterGroup) Delete(path string, handler core.Handler) *Route {
	route := &Route{Method: "DELETE", Path: g.prefix + path}
	g.add([]string{"DELETE"}, path, handler, route)
	return route
}

func (g *RouterGroup) All(path string, handler core.Handler) *Route {
	route := &Route{Method: "ALL", Path: g.prefix + path}
	g.add([]string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}, path, handler, route)
	return route
}
