package router

import (
	"strings"
	"sync"
	"time"

	"github.com/Dziqha/TurboGo/core"
)

// ===== OPTIMIZED STRUCTURES =====

// Reduced memory footprint by removing unused fields
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

// More memory-efficient route structures
type WildcardRoute struct {
	prefix  string
	handler core.Handler
	route   *Route
	pLen    uint8 // Changed from int to uint8 (saves 7 bytes per route)
}

type ParametricRoute struct {
	pattern  string
	handler  core.Handler
	route    *Route
	params   []string
	segments []string
}

// Removed unused Connection struct
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
	// Reduced initial map sizes to prevent over-allocation
	staticGET    map[string]*FastRoute
	staticPOST   map[string]*FastRoute
	staticPUT    map[string]*FastRoute
	staticDELETE map[string]*FastRoute
	staticOther  map[string]*FastRoute

	// Use arrays instead of slices for better memory locality
	wildcardGET    []WildcardRoute
	wildcardPOST   []WildcardRoute
	wildcardPUT    []WildcardRoute
	wildcardDELETE []WildcardRoute
	wildcardOther  []WildcardRoute

	parametricGET    []ParametricRoute
	parametricPOST   []ParametricRoute
	parametricPUT    []ParametricRoute
	parametricDELETE []ParametricRoute
	parametricOther  []ParametricRoute

	// Removed unused pools and buffers
	notFoundHandler core.Handler
	config          RouterConfig
}

func NewSingleThreadedRouter() *SingleThreadedRouter {
	return &SingleThreadedRouter{
		// Start with minimal sizes and let Go grow as needed
		staticGET:    make(map[string]*FastRoute, 8),
		staticPOST:   make(map[string]*FastRoute, 4),
		staticPUT:    make(map[string]*FastRoute, 2),
		staticDELETE: make(map[string]*FastRoute, 2),
		staticOther:  make(map[string]*FastRoute, 2),

		// Minimal initial capacity
		wildcardGET:    make([]WildcardRoute, 0, 2),
		wildcardPOST:   make([]WildcardRoute, 0, 1),
		wildcardPUT:    make([]WildcardRoute, 0, 1),
		wildcardDELETE: make([]WildcardRoute, 0, 1),
		wildcardOther:  make([]WildcardRoute, 0, 1),

		parametricGET:    make([]ParametricRoute, 0, 4),
		parametricPOST:   make([]ParametricRoute, 0, 2),
		parametricPUT:    make([]ParametricRoute, 0, 2),
		parametricDELETE: make([]ParametricRoute, 0, 2),
		parametricOther:  make([]ParametricRoute, 0, 2),

		notFoundHandler: func(c *core.Context) {
			c.Status(404).SendString("404 Not Found")
		},
		config: DefaultRouterConfig(),
	}
}

func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		MaxCacheSize:       1000, // Reduced from 10000
		EvictionPolicy:     "lru",
		EnableMetrics:      false, // Disabled by default
		EnableCaching:      false, // Disabled by default
		EnablePooling:      false, // Disabled by default
		EnableUnsafeOpt:    true,
		StaticPreAlloc:     64, // Reduced from 4096
		WildcardPreAlloc:   8,  // Reduced from 256
		ParametricPreAlloc: 16, // Reduced from 256
	}
}

// Optimized prefix matching without unsafe operations for better stability
func quickPrefix(s, prefix string, pLen int) bool {
	if len(s) < pLen {
		return false
	}
	return s[:pLen] == prefix
}

// Memory-efficient parametric matching using sync.Pool for parameter maps
var paramMapPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]string, 4)
	},
}

func (r *SingleThreadedRouter) matchParametric(param *ParametricRoute, path string) (bool, map[string]string) {
	pathSegments := strings.Split(path, "/")

	if len(pathSegments) != len(param.segments) {
		return false, nil
	}

	// Get from pool instead of allocating
	paramMap := paramMapPool.Get().(map[string]string)

	// Clear the map
	for k := range paramMap {
		delete(paramMap, k)
	}

	paramIdx := 0
	for i, segment := range param.segments {
		if len(segment) > 0 && segment[0] == ':' {
			if paramIdx < len(param.params) {
				paramMap[param.params[paramIdx]] = pathSegments[i]
				paramIdx++
			}
		} else if segment != pathSegments[i] {
			// Return to pool before returning false
			paramMapPool.Put(paramMap)
			return false, nil
		}
	}

	if len(paramMap) == 0 {
		paramMapPool.Put(paramMap)
		return true, nil
	}

	// Create result map and return the pooled one
	result := make(map[string]string, len(paramMap))
	for k, v := range paramMap {
		result[k] = v
	}
	paramMapPool.Put(paramMap)

	return true, result
}

// ===== ROUTE REGISTRATION =====

func (r *SingleThreadedRouter) AddStatic(method, path string, handler core.Handler, route *Route) {
	fastRoute := &FastRoute{
		handler: handler,
		route:   route,
	}

	switch method {
	case "GET":
		r.staticGET[path] = fastRoute
	case "POST":
		r.staticPOST[path] = fastRoute
	case "PUT":
		r.staticPUT[path] = fastRoute
	case "DELETE":
		r.staticDELETE[path] = fastRoute
	default:
		r.staticOther[method+":"+path] = fastRoute
	}
}

func (r *SingleThreadedRouter) AddWildcard(method, prefix string, handler core.Handler, route *Route) {
	wildRoute := WildcardRoute{
		prefix:  prefix,
		handler: handler,
		route:   route,
		pLen:    uint8(len(prefix)), // Use uint8 instead of int
	}

	switch method {
	case "GET":
		r.wildcardGET = append(r.wildcardGET, wildRoute)
	case "POST":
		r.wildcardPOST = append(r.wildcardPOST, wildRoute)
	case "PUT":
		r.wildcardPUT = append(r.wildcardPUT, wildRoute)
	case "DELETE":
		r.wildcardDELETE = append(r.wildcardDELETE, wildRoute)
	default:
		r.wildcardOther = append(r.wildcardOther, wildRoute)
	}
}

func (r *SingleThreadedRouter) AddParametric(method, pattern string, handler core.Handler, route *Route) {
	segments := strings.Split(pattern, "/")
	params := make([]string, 0, 2) // Pre-allocate with small capacity

	for _, segment := range segments {
		if len(segment) > 0 && segment[0] == ':' {
			params = append(params, segment[1:])
		}
	}

	paramRoute := ParametricRoute{
		pattern:  pattern,
		handler:  handler,
		route:    route,
		params:   params,
		segments: segments,
	}

	switch method {
	case "GET":
		r.parametricGET = append(r.parametricGET, paramRoute)
	case "POST":
		r.parametricPOST = append(r.parametricPOST, paramRoute)
	case "PUT":
		r.parametricPUT = append(r.parametricPUT, paramRoute)
	case "DELETE":
		r.parametricDELETE = append(r.parametricDELETE, paramRoute)
	default:
		r.parametricOther = append(r.parametricOther, paramRoute)
	}
}

// ===== OPTIMIZED ROUTE LOOKUP =====

func (r *SingleThreadedRouter) Find(method, path string) (core.Handler, *Route, map[string]string) {
	// GET optimization (most common case first)
	if method == "GET" {
		if route := r.staticGET[path]; route != nil {
			return route.handler, route.route, nil
		}

		// Wildcard routes
		for i := range r.wildcardGET {
			wild := &r.wildcardGET[i]
			if len(path) >= int(wild.pLen) && quickPrefix(path, wild.prefix, int(wild.pLen)) {
				return wild.handler, wild.route, nil
			}
		}

		// Parametric routes
		for i := range r.parametricGET {
			param := &r.parametricGET[i]
			if matched, params := r.matchParametric(param, path); matched {
				return param.handler, param.route, params
			}
		}

		return r.notFoundHandler, nil, nil
	}

	// Handle other methods efficiently
	var staticMap map[string]*FastRoute
	var wildcards []WildcardRoute
	var params []ParametricRoute

	switch method {
	case "POST":
		staticMap, wildcards, params = r.staticPOST, r.wildcardPOST, r.parametricPOST
	case "PUT":
		staticMap, wildcards, params = r.staticPUT, r.wildcardPUT, r.parametricPUT
	case "DELETE":
		staticMap, wildcards, params = r.staticDELETE, r.wildcardDELETE, r.parametricDELETE
	default:
		staticMap, wildcards, params = r.staticOther, r.wildcardOther, r.parametricOther
		// Try method-prefixed lookup for other methods
		if route := r.staticOther[method+":"+path]; route != nil {
			return route.handler, route.route, nil
		}
	}

	// Static lookup
	if route := staticMap[path]; route != nil {
		return route.handler, route.route, nil
	}

	// Wildcard lookup
	for i := range wildcards {
		wild := &wildcards[i]
		if len(path) >= int(wild.pLen) && quickPrefix(path, wild.prefix, int(wild.pLen)) {
			return wild.handler, wild.route, nil
		}
	}

	// Parametric lookup
	for i := range params {
		param := &params[i]
		if matched, paramMap := r.matchParametric(param, path); matched {
			return param.handler, param.route, paramMap
		}
	}

	return r.notFoundHandler, nil, nil
}

// ===== UTILITY METHODS =====

func (r *SingleThreadedRouter) Reset() {
	// More efficient map clearing
	r.staticGET = make(map[string]*FastRoute, 8)
	r.staticPOST = make(map[string]*FastRoute, 4)
	r.staticPUT = make(map[string]*FastRoute, 2)
	r.staticDELETE = make(map[string]*FastRoute, 2)
	r.staticOther = make(map[string]*FastRoute, 2)

	// Reset slices to zero length but keep capacity
	r.wildcardGET = r.wildcardGET[:0]
	r.wildcardPOST = r.wildcardPOST[:0]
	r.wildcardPUT = r.wildcardPUT[:0]
	r.wildcardDELETE = r.wildcardDELETE[:0]
	r.wildcardOther = r.wildcardOther[:0]

	r.parametricGET = r.parametricGET[:0]
	r.parametricPOST = r.parametricPOST[:0]
	r.parametricPUT = r.parametricPUT[:0]
	r.parametricDELETE = r.parametricDELETE[:0]
	r.parametricOther = r.parametricOther[:0]
}

func (r *SingleThreadedRouter) GetRouteCount() (static, wildcard, parametric int) {
	static = len(r.staticGET) + len(r.staticPOST) + len(r.staticPUT) + len(r.staticDELETE) + len(r.staticOther)
	wildcard = len(r.wildcardGET) + len(r.wildcardPOST) + len(r.wildcardPUT) + len(r.wildcardDELETE) + len(r.wildcardOther)
	parametric = len(r.parametricGET) + len(r.parametricPOST) + len(r.parametricPUT) + len(r.parametricDELETE) + len(r.parametricOther)
	return
}

// Simplified warm-up without unnecessary allocations
func (r *SingleThreadedRouter) WarmUp(routes []struct{ Method, Path string }) {
	for _, route := range routes {
		r.Find(route.Method, route.Path)
	}
}

// // Route configuration methods
// func (r *Route) NoCache() *Route {
// 	r.Options.Disable = true
// 	return r
// }

// func (r *Route) Cache(ttl time.Duration) *Route {
// 	r.Options.Ttl = &ttl
// 	r.Options.Disable = false
// 	r.Options.Force = true
// 	return r
// }

// func (r *Route) Named(name string) *Route {
// 	r.Name = name
// 	return r
// }
