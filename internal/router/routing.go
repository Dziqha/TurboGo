package router

import (
	"strings"
	"time"

	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

type Route struct {
	Method   string
	Path     string
	Name     string
	Handlers   []core.Handler
	Options    RouteOptions
	FastHandler fasthttp.RequestHandler
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

	// compiled is the transcribed switch table (see compile.go). It is only
	// consulted when compiledOn is true; the tree remains the fallback for
	// anything the table cannot express. dirty marks the table stale after
	// route changes (lazy recompile on next compiled lookup).
	compiled   *Compiled
	compiledOn bool
	dirty      bool

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
	r.markDirty()
}

func (r *SingleThreadedRouter) AddStaticFast(method, path string, fastHandler fasthttp.RequestHandler) *Route {
	route := &Route{
		Method:      method,
		Path:        path,
		FastHandler: fastHandler,
	}
	switch method {
	case "GET":
		r.staticGET[path] = &FastRoute{handler: nil, route: route}
	case "POST":
		r.staticPOST[path] = &FastRoute{handler: nil, route: route}
	case "PUT":
		r.staticPUT[path] = &FastRoute{handler: nil, route: route}
	case "DELETE":
		r.staticDELETE[path] = &FastRoute{handler: nil, route: route}
	default:
		r.staticOther[method+":"+path] = &FastRoute{handler: nil, route: route}
	}
	r.markDirty()
	return route
}

func (r *SingleThreadedRouter) AddWildcard(method, prefix string, handler core.Handler, route *Route) {
	r.tree.insert(method, prefix+"*", handler, route)
	r.markDirty()
}

func (r *SingleThreadedRouter) AddParametric(method, pattern string, handler core.Handler, route *Route) {
	r.tree.insert(method, pattern, handler, route)
	r.markDirty()
}

// markDirty flags the compiled table stale. The table rebuilds lazily on
// the next compiled lookup, so bulk registration pays one recompile.
func (r *SingleThreadedRouter) markDirty() {
	if r.compiledOn {
		r.dirty = true
	}
}

func (r *SingleThreadedRouter) Find(method, path string, params map[string]string) (core.Handler, *Route) {
	if method == "HEAD" {
		if h, rt := r.find("HEAD", path, params); rt != nil {
			return h, rt
		}
		if h, rt := r.find("GET", path, params); rt != nil {
			return h, rt
		}
		return r.notFoundOrNotAllowed(path)
	}
	return r.find(method, path, params)
}

// FindInto is the zero-alloc lookup used on the request hot path. Matched
// params land directly in dst slots (no hashing); the return value is their
// count. overflow carries anything that cannot fit slots: tree-fallback
// matches and routes with more than core.MaxSlots params. Static,
// 404 and 405 hits write nothing. Callers must honor n even when the
// overflow map is non-empty (both can hold values after a fallback match).
func (r *SingleThreadedRouter) FindInto(method, path string, dst *[core.MaxSlots]core.ParamSlot, overflow map[string]string) (core.Handler, *Route, int) {
	if method == "HEAD" {
		if h, rt, n := r.findInto("HEAD", path, dst, overflow); rt != nil {
			return h, rt, n
		}
		// HEAD falls back to GET: retry into clean sinks.
		if overflow != nil {
			for k := range overflow {
				delete(overflow, k)
			}
		}
		return r.findInto("GET", path, dst, overflow)
	}
	return r.findInto(method, path, dst, overflow)
}

func (r *SingleThreadedRouter) findInto(method, path string, dst *[core.MaxSlots]core.ParamSlot, overflow map[string]string) (core.Handler, *Route, int) {
	// mi mirrors methodIdx; treeOnly forces tree search for unknown methods
	// (tree aliases them to the GET slot, which the tables don't replicate).
	mi := 0
	treeOnly := false
	switch method {
	case "GET":
		if fr := r.staticGET[path]; fr != nil {
			return fr.handler, fr.route, 0
		}
		mi = 0
	case "POST":
		if fr := r.staticPOST[path]; fr != nil {
			return fr.handler, fr.route, 0
		}
		mi = 1
	case "PUT":
		if fr := r.staticPUT[path]; fr != nil {
			return fr.handler, fr.route, 0
		}
		mi = 2
	case "DELETE":
		if fr := r.staticDELETE[path]; fr != nil {
			return fr.handler, fr.route, 0
		}
		mi = 3
	default:
		if fr := r.staticOther[method+":"+path]; fr != nil {
			return fr.handler, fr.route, 0
		}
		var ok bool
		mi, ok = methodIndex(method)
		treeOnly = !ok
	}

	if r.compiledOn && !treeOnly {
		if r.compiled != nil {
			return r.findIntoCompiled(method, mi, path, dst, overflow)
		}
	}

	h, rt, ok := r.tree.search(method, path, overflow)
	if ok {
		// Tree writes the overflow map directly; slots stay empty.
		// Callers consult both (Param checks slots, then map).
		return h, rt, 0
	}
	h, rt = r.notFoundOrNotAllowed(path)
	return h, rt, 0
}

func (r *SingleThreadedRouter) find(method, path string, params map[string]string) (core.Handler, *Route) {
	// mi mirrors methodIdx; treeOnly forces tree search for unknown methods
	// (tree aliases them to the GET slot, which the tables don't replicate).
	mi := 0
	treeOnly := false
	switch method {
	case "GET":
		if fr := r.staticGET[path]; fr != nil {
			return fr.handler, fr.route
		}
		mi = 0
	case "POST":
		if fr := r.staticPOST[path]; fr != nil {
			return fr.handler, fr.route
		}
		mi = 1
	case "PUT":
		if fr := r.staticPUT[path]; fr != nil {
			return fr.handler, fr.route
		}
		mi = 2
	case "DELETE":
		if fr := r.staticDELETE[path]; fr != nil {
			return fr.handler, fr.route
		}
		mi = 3
	default:
		if fr := r.staticOther[method+":"+path]; fr != nil {
			return fr.handler, fr.route
		}
		var ok bool
		mi, ok = methodIndex(method)
		treeOnly = !ok
	}

	if r.compiledOn && r.compiled != nil && !treeOnly {
		return r.findCompiled(method, mi, path, params)
	}

	handler, route, ok := r.tree.search(method, path, params)
	if ok {
		return handler, route
	}

	return r.notFoundOrNotAllowed(path)
}

// findIntoCompiled mirrors findCompiled with slot sinks: the decision
// search fills stack offsets, and only the winner is materialized —
// directly into dst, never through a map.
func (r *SingleThreadedRouter) findIntoCompiled(method string, mi int, path string, dst *[core.MaxSlots]core.ParamSlot, overflow map[string]string) (core.Handler, *Route, int) {
	if r.dirty {
		r.compiled = r.Compile()
		r.dirty = false
	}
	comp := r.compiled
	var bestP *cLeaf
	var slots [maxSegs][2]int
	if comp.root != nil {
		// pos=1 mirrors tree.search entry (path[0] is '/' in practice).
		leaf, ovf := comp.root.search(path, 1, 0, mi, &slots)
		if ovf {
			// Path too deep for slots: nothing was written anywhere.
			if h, rt, ok := r.tree.search(method, path, overflow); ok {
				return h, rt, 0
			}
		} else if leaf != nil {
			bestP = leaf
		}
	}

	var bestW *cWildcard
	for _, w := range comp.wild[mi] {
		if len(path) > len(w.prefix) && strings.HasPrefix(path, w.prefix) {
			bestW = w
			break
		}
	}

	if bestP != nil && (bestW == nil || bestP.ord < bestW.ord) {
		if len(bestP.names) > len(dst) {
			// More params than slots: exact check via tree into overflow.
			if h, rt, ok := r.tree.search(method, path, overflow); ok {
				return h, rt, 0
			}
		} else {
			for i, si := range bestP.pdepths {
				s, e := slots[si][0], slots[si][1]
				dst[i] = core.ParamSlot{K: bestP.names[i], V: path[s:e]}
			}
			return bestP.handler, bestP.route, len(bestP.names)
		}
	}
	if bestW != nil {
		dst[0] = core.ParamSlot{K: "*", V: stripSlashPrefix(path[len(bestW.prefix):])}
		return bestW.handler, bestW.route, 1
	}
	// Miss: fall back to the tree for an exact answer.
	if h, rt, ok := r.tree.search(method, path, overflow); ok {
		return h, rt, 0
	}
	h, rt := r.notFoundOrNotAllowed(path)
	return h, rt, 0
}

// stripSlashPrefix mirrors the tree wildcard capture: at most one leading
// slash is skipped before capturing the remainder.
func stripSlashPrefix(rest string) string {
	if len(rest) > 0 && rest[0] == '/' {
		rest = rest[1:]
	}
	return rest
}
func (r *SingleThreadedRouter) findCompiled(method string, mi int, path string, params map[string]string) (core.Handler, *Route) {
	if r.dirty {
		r.compiled = r.Compile()
		r.dirty = false
	}
	var bestP *cLeaf
	var slots [maxSegs][2]int
	if r.compiled.root != nil {
		// pos=1 mirrors tree.search entry (path[0] is '/' in practice).
		leaf, ovf := r.compiled.root.search(path, 1, 0, mi, &slots)
		if ovf {
			// Path too deep for slots: nothing was written to params.
			if h, rt, ok := r.tree.search(method, path, params); ok {
				return h, rt
			}
		} else if leaf != nil {
			bestP = leaf
		}
	}

	var bestW *cWildcard
	for _, w := range r.compiled.wild[mi] {
		if len(path) > len(w.prefix) && strings.HasPrefix(path, w.prefix) {
			bestW = w
			break
		}
	}

	if bestP != nil && (bestW == nil || bestP.ord < bestW.ord) {
		// Copy slots only for the winner: loser branches leave no keys,
		// mirroring the tree's short-circuit.
		if params != nil {
			for i, si := range bestP.pdepths {
				s, e := slots[si][0], slots[si][1]
				params[bestP.names[i]] = path[s:e]
			}
		}
		return bestP.handler, bestP.route
	}
	if bestW != nil {
		if params != nil {
			// Decision writes nothing until a win, so no loser keys exist.
			rest := path[len(bestW.prefix):]
			if len(rest) > 0 && rest[0] == '/' {
				rest = rest[1:]
			}
			params["*"] = rest
		}
		return bestW.handler, bestW.route
	}
	// Miss: fall back to the tree for an exact answer.
	if h, rt, ok := r.tree.search(method, path, params); ok {
		return h, rt
	}
	return r.notFoundOrNotAllowed(path)
}

func (r *SingleThreadedRouter) notFoundOrNotAllowed(path string) (core.Handler, *Route) {
	if r.tree.pathExistsAny(path) {
		return r.methodNotAllowedHandler, nil
	}
	return r.notFoundHandler, nil
}

func (r *SingleThreadedRouter) SetNotFoundHandler(h core.Handler) {
	if h != nil {
		r.notFoundHandler = h
	}
}

func (r *SingleThreadedRouter) SetMethodNotAllowedHandler(h core.Handler) {
	if h != nil {
		r.methodNotAllowedHandler = h
	}
}

func (r *SingleThreadedRouter) AddFastRoute(method, path string, fastHandler fasthttp.RequestHandler) *Route {
	route := &Route{
		Method:      method,
		Path:        path,
		FastHandler: fastHandler,
	}
	r.tree.insert(method, path, nil, route)
	r.markDirty()
	return route
}

func (r *SingleThreadedRouter) Reset() {
	r.staticGET = make(map[string]*FastRoute, 8)
	r.staticPOST = make(map[string]*FastRoute, 4)
	r.staticPUT = make(map[string]*FastRoute, 2)
	r.staticDELETE = make(map[string]*FastRoute, 2)
	r.staticOther = make(map[string]*FastRoute, 2)
	r.tree = newRadixTree()
	r.markDirty()
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
