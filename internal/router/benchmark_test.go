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

// ===== Large-scale bench (~200 routes, GitHub-API style) =====
// Baseline oracle for the compiled-switch router (S3 must beat these).

var scaleStaticPaths []string
var scaleParamPaths [][2]string // {pattern, sample}

func init() {
	prefixes := []string{"users", "repos", "orgs", "teams", "issues", "pulls", "actions", "packages", "gists", "projects"}
	for _, p := range prefixes {
		for i := 0; i < 10; i++ {
			scaleStaticPaths = append(scaleStaticPaths, "/api/"+p+"/list"+itoa(i))
		}
	}
	for _, p := range prefixes {
		scaleParamPaths = append(scaleParamPaths,
			[2]string{"/api/" + p + "/:id", "/api/" + p + "/42"},
			[2]string{"/api/" + p + "/:id/posts", "/api/" + p + "/42/posts"},
		)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [8]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

func prepareScaleRouter() *SingleThreadedRouter {
	r := NewSingleThreadedRouter()
	for _, p := range scaleStaticPaths {
		r.AddStatic("GET", p, dummyHandler, &Route{Path: p})
	}
	for _, pp := range scaleParamPaths {
		r.AddParametric("GET", pp[0], dummyHandler, &Route{Path: pp[0]})
	}
	r.AddStatic("GET", "/users/new", dummyHandler, &Route{Path: "/users/new"})
	r.AddParametric("GET", "/users/:id", dummyHandler, &Route{Path: "/users/:id"})
	return r
}

func BenchmarkRouter_ScaleStatic(b *testing.B) {
	r := prepareScaleRouter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find("GET", scaleStaticPaths[i%len(scaleStaticPaths)], nil)
	}
}

func BenchmarkRouter_ScaleParam(b *testing.B) {
	r := prepareScaleRouter()
	p := make(map[string]string)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pp := scaleParamPaths[i%len(scaleParamPaths)]
		r.Find("GET", pp[1], p)
		for k := range p {
			delete(p, k)
		}
	}
}

func BenchmarkRouter_ScaleStaticBeatsParam(b *testing.B) {
	r := prepareScaleRouter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find("GET", "/users/new", nil)
	}
}

// ===== Compiled-switch variants (flag on) =====

func prepareCompiledRouter() *SingleThreadedRouter {
	r := prepareScaleRouter()
	r.SetCompiled(true)
	return r
}

func BenchmarkRouter_CompiledParam(b *testing.B) {
	r := prepareCompiledRouter()
	p := make(map[string]string)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pp := scaleParamPaths[i%len(scaleParamPaths)]
		r.Find("GET", pp[1], p)
		for k := range p {
			delete(p, k)
		}
	}
}

func BenchmarkRouter_CompiledWildcard(b *testing.B) {
	r := prepareBenchRouter()
	r.SetCompiled(true)
	p := make(map[string]string)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find("GET", "/assets/images/logo.png", p)
		delete(p, "*")
	}
}

func BenchmarkRouter_CompiledStatic(b *testing.B) {
	r := prepareCompiledRouter()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Find("GET", scaleStaticPaths[i%len(scaleStaticPaths)], nil)
	}
}
