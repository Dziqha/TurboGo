package router

import "testing"

var benchPaths = []string{
	"/",
	"/api/health",
	"/users/42",
	"/assets/images/logo.png",
	"/api/v1/users",
	"/users/99/posts",
}

func prepareBenchRouter() *SingleThreadedRouter {
	r := NewSingleThreadedRouter()
	r.AddStatic("GET", "/", dummyHandler, &Route{Path: "/"})
	r.AddStatic("GET", "/api/health", dummyHandler, &Route{Path: "/api/health"})
	r.AddParametric("GET", "/users/:id", dummyHandler, &Route{Path: "/users/:id"})
	r.AddWildcard("GET", "/assets/", dummyHandler, &Route{Path: "/assets/*"})
	r.AddParametric("GET", "/api/:version/users", dummyHandler, &Route{Path: "/api/:version/users"})
	r.AddParametric("GET", "/users/:id/posts", dummyHandler, &Route{Path: "/users/:id/posts"})
	return r
}

func BenchmarkRouter_Static(b *testing.B) {
	r := prepareBenchRouter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find("GET", "/api/health", nil)
	}
}

func BenchmarkRouter_Param(b *testing.B) {
	r := prepareBenchRouter()
	p := make(map[string]string)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find("GET", "/users/42", p)
		delete(p, "id")
	}
}

func BenchmarkRouter_Wildcard(b *testing.B) {
	r := prepareBenchRouter()
	p := make(map[string]string)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find("GET", "/assets/images/logo.png", p)
		delete(p, "*")
	}
}

func BenchmarkRouter_MultipleRoutes(b *testing.B) {
	r := prepareBenchRouter()
	p := make(map[string]string)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % len(benchPaths)
		switch benchPaths[idx] {
		case "/":
			r.Find("GET", benchPaths[idx], nil)
		case "/api/health":
			r.Find("GET", benchPaths[idx], nil)
		default:
			r.Find("GET", benchPaths[idx], p)
			for k := range p {
				delete(p, k)
			}
		}
	}
}
