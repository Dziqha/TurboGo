package router

import (
	"testing"

	"github.com/Dziqha/TurboGo/core"
	"github.com/stretchr/testify/assert"
)

func dummyHandler(c *core.Context) {}

func TestRadixTree_StaticRoutes(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/", dummyHandler, &Route{Path: "/", Method: "GET"})
	tree.insert("GET", "/users", dummyHandler, &Route{Path: "/users", Method: "GET"})
	tree.insert("GET", "/users/profile", dummyHandler, &Route{Path: "/users/profile", Method: "GET"})

	h, r, ok := tree.search("GET", "/", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/", r.Path)

	h, r, ok = tree.search("GET", "/users", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users", r.Path)

	h, r, ok = tree.search("GET", "/users/profile", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/profile", r.Path)

	h, _, ok = tree.search("GET", "/nonexistent", nil)
	assert.False(t, ok)
	assert.Nil(t, h)
}

func TestRadixTree_ParametricRoutes(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/users/:id", dummyHandler, &Route{Path: "/users/:id", Method: "GET"})
	tree.insert("GET", "/users/:id/posts", dummyHandler, &Route{Path: "/users/:id/posts", Method: "GET"})
	tree.insert("GET", "/api/:version/:resource", dummyHandler, &Route{Path: "/api/:version/:resource", Method: "GET"})

	p := map[string]string{}
	h, r, ok := tree.search("GET", "/users/42", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/:id", r.Path)
	assert.Equal(t, "42", p["id"])

	p = map[string]string{}
	h, r, ok = tree.search("GET", "/users/99/posts", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/:id/posts", r.Path)
	assert.Equal(t, "99", p["id"])

	p = map[string]string{}
	h, r, ok = tree.search("GET", "/api/v2/orders", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/api/:version/:resource", r.Path)
	assert.Equal(t, "v2", p["version"])
	assert.Equal(t, "orders", p["resource"])
}

func TestRadixTree_WildcardRoutes(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/static/*", dummyHandler, &Route{Path: "/static/*", Method: "GET"})
	tree.insert("GET", "/static/sub/*", dummyHandler, &Route{Path: "/static/sub/*", Method: "GET"})

	p := map[string]string{}
	h, r, ok := tree.search("GET", "/static/css/app.css", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/static/*", r.Path)
	assert.Equal(t, "css/app.css", p["*"])

	p = map[string]string{}
	h, r, ok = tree.search("GET", "/static/sub/file.js", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/static/sub/*", r.Path)
	assert.Equal(t, "file.js", p["*"])

	h, _, ok = tree.search("GET", "/other/path", nil)
	assert.False(t, ok)
	assert.Nil(t, h)
}

func TestRadixTree_MixedRoutes(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/", dummyHandler, &Route{Path: "/", Method: "GET"})
	tree.insert("GET", "/api/health", dummyHandler, &Route{Path: "/api/health", Method: "GET"})
	tree.insert("GET", "/api/:version/users", dummyHandler, &Route{Path: "/api/:version/users", Method: "GET"})
	tree.insert("GET", "/assets/*", dummyHandler, &Route{Path: "/assets/*", Method: "GET"})

	h, r, ok := tree.search("GET", "/", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/", r.Path)

	h, r, ok = tree.search("GET", "/api/health", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/api/health", r.Path)

	p := map[string]string{}
	h, r, ok = tree.search("GET", "/api/v1/users", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/api/:version/users", r.Path)
	assert.Equal(t, "v1", p["version"])

	p = map[string]string{}
	h, r, ok = tree.search("GET", "/assets/images/logo.png", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/assets/*", r.Path)
	assert.Equal(t, "images/logo.png", p["*"])

	h, _, ok = tree.search("GET", "/api/v1/other", nil)
	assert.False(t, ok)
	assert.Nil(t, h)
}

func TestRadixTree_ParamPriority(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/users/me", dummyHandler, &Route{Path: "/users/me", Method: "GET"})
	tree.insert("GET", "/users/:id", dummyHandler, &Route{Path: "/users/:id", Method: "GET"})

	h, r, ok := tree.search("GET", "/users/me", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/me", r.Path)

	p := map[string]string{}
	h, r, ok = tree.search("GET", "/users/42", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/:id", r.Path)
	assert.Equal(t, "42", p["id"])
}

func TestRadixTree_CommonPrefix(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/user", dummyHandler, &Route{Path: "/user", Method: "GET"})
	tree.insert("GET", "/users", dummyHandler, &Route{Path: "/users", Method: "GET"})
	tree.insert("GET", "/users/:id", dummyHandler, &Route{Path: "/users/:id", Method: "GET"})

	h, r, ok := tree.search("GET", "/user", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/user", r.Path)

	h, r, ok = tree.search("GET", "/users", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users", r.Path)

	p := map[string]string{}
	h, r, ok = tree.search("GET", "/users/99", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/:id", r.Path)
	assert.Equal(t, "99", p["id"])
}

func TestRadixTree_DeeplyNested(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/a/b/c/d/e", dummyHandler, &Route{Path: "/a/b/c/d/e"})

	h, r, ok := tree.search("GET", "/a/b/c/d/e", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/a/b/c/d/e", r.Path)

	h, _, ok = tree.search("GET", "/a/b/c", nil)
	assert.False(t, ok)
	assert.Nil(t, h)
}

func TestRadixTree_ParamAndStaticAtSameLevel(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/users/me", dummyHandler, &Route{Path: "/users/me"})
	tree.insert("GET", "/users/:id", dummyHandler, &Route{Path: "/users/:id"})

	h, r, ok := tree.search("GET", "/users/me", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/me", r.Path)

	p := map[string]string{}
	h, r, ok = tree.search("GET", "/users/42", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/:id", r.Path)
	assert.Equal(t, "42", p["id"])
}

func TestRadixTree_EmptyPath(t *testing.T) {
	tree := newRadixTree()

	h, _, ok := tree.search("GET", "/", nil)
	assert.False(t, ok)
	assert.Nil(t, h)

	tree.insert("GET", "/", dummyHandler, &Route{Path: "/"})
	h, r, ok := tree.search("GET", "/", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/", r.Path)
}

func TestRadixTree_ParamWithNestedStatic(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/api/:version/health", dummyHandler, &Route{Path: "/api/:version/health"})

	p := map[string]string{}
	h, r, ok := tree.search("GET", "/api/v1/health", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/api/:version/health", r.Path)
	assert.Equal(t, "v1", p["version"])

	h, _, ok = tree.search("GET", "/api/v1/status", nil)
	assert.False(t, ok)
	assert.Nil(t, h)
}

func TestRadixTree_MultipleParams(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/:a/:b/:c", dummyHandler, &Route{Path: "/:a/:b/:c"})

	p := map[string]string{}
	h, _, ok := tree.search("GET", "/x/y/z", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "x", p["a"])
	assert.Equal(t, "y", p["b"])
	assert.Equal(t, "z", p["c"])
}

func TestRadixTree_WildcardAtRoot(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/*", dummyHandler, &Route{Path: "/*"})

	p := map[string]string{}
	h, _, ok := tree.search("GET", "/anything/at/all", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "anything/at/all", p["*"])

	p = map[string]string{}
	h, _, ok = tree.search("GET", "/single", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "single", p["*"])
}

func TestRadixTree_ParamAfterWildcard_StaticPreferred(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/static/*", dummyHandler, &Route{Path: "/static/*"})
	tree.insert("GET", "/static/exact", dummyHandler, &Route{Path: "/static/exact"})

	h, r, ok := tree.search("GET", "/static/exact", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/static/exact", r.Path)

	p := map[string]string{}
	h, r, ok = tree.search("GET", "/static/other/file.txt", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/static/*", r.Path)
	assert.Equal(t, "other/file.txt", p["*"])
}

func TestRadixTree_MethodIsolation(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/data", dummyHandler, &Route{Path: "/data", Method: "GET"})
	tree.insert("POST", "/data", dummyHandler, &Route{Path: "/data", Method: "POST"})

	h, r, ok := tree.search("GET", "/data", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "GET", r.Method)

	h, r, ok = tree.search("POST", "/data", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "POST", r.Method)

	h, _, ok = tree.search("DELETE", "/data", nil)
	assert.False(t, ok)
	assert.Nil(t, h)

	tree.insert("DELETE", "/data", dummyHandler, &Route{Path: "/data", Method: "DELETE"})
	h, r, ok = tree.search("DELETE", "/data", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "DELETE", r.Method)
}

func TestRadixTree_ParamMatches(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/posts/:id", dummyHandler, &Route{Path: "/posts/:id"})

	p := map[string]string{}
	h, _, ok := tree.search("GET", "/posts/new", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "new", p["id"])
}

// SingleThreadedRouter integration tests

func TestRouter_Static(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddStatic("GET", "/", dummyHandler, &Route{Path: "/", Method: "GET"})
	r.AddStatic("GET", "/hello", dummyHandler, &Route{Path: "/hello", Method: "GET"})

	h, route := r.Find("GET", "/", nil)
	assert.NotNil(t, h)
	assert.Equal(t, "/", route.Path)

	h, route = r.Find("GET", "/hello", nil)
	assert.NotNil(t, h)
	assert.Equal(t, "/hello", route.Path)
}

func TestRouter_Parametric(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddParametric("GET", "/users/:id", dummyHandler, &Route{Path: "/users/:id", Method: "GET"})

	p := map[string]string{}
	h, route := r.Find("GET", "/users/42", p)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/:id", route.Path)
	assert.Equal(t, "42", p["id"])
}

func TestRouter_Wildcard(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddWildcard("GET", "/static/", dummyHandler, &Route{Path: "/static/*", Method: "GET"})

	p := map[string]string{}
	h, route := r.Find("GET", "/static/css/app.css", p)
	assert.NotNil(t, h)
	assert.Equal(t, "/static/*", route.Path)
	assert.Equal(t, "css/app.css", p["*"])
}

func TestRouter_Mixed(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddStatic("GET", "/", dummyHandler, &Route{Path: "/", Method: "GET"})
	r.AddStatic("GET", "/api/health", dummyHandler, &Route{Path: "/api/health", Method: "GET"})
	r.AddParametric("GET", "/users/:id", dummyHandler, &Route{Path: "/users/:id", Method: "GET"})
	r.AddWildcard("GET", "/assets/", dummyHandler, &Route{Path: "/assets/*", Method: "GET"})

	h, route := r.Find("GET", "/", nil)
	assert.NotNil(t, h)
	assert.Equal(t, "/", route.Path)

	h, route = r.Find("GET", "/api/health", nil)
	assert.NotNil(t, h)
	assert.Equal(t, "/api/health", route.Path)

	p := map[string]string{}
	h, route = r.Find("GET", "/users/99", p)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/:id", route.Path)
	assert.Equal(t, "99", p["id"])

	p = map[string]string{}
	h, route = r.Find("GET", "/assets/images/logo.png", p)
	assert.NotNil(t, h)
	assert.Equal(t, "/assets/*", route.Path)
	assert.Equal(t, "images/logo.png", p["*"])
}

func TestRouter_NotFoundHandler(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddStatic("GET", "/exists", dummyHandler, &Route{Path: "/exists", Method: "GET"})

	h, _ := r.Find("GET", "/nonexistent", nil)
	assert.NotNil(t, h)
}

func TestRouter_PostMethod(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddStatic("POST", "/data", dummyHandler, &Route{Path: "/data", Method: "POST"})
	r.AddParametric("POST", "/items/:id", dummyHandler, &Route{Path: "/items/:id", Method: "POST"})

	h, route := r.Find("POST", "/data", nil)
	assert.NotNil(t, h)
	assert.Equal(t, "/data", route.Path)

	p := map[string]string{}
	h, route = r.Find("POST", "/items/5", p)
	assert.NotNil(t, h)
	assert.Equal(t, "/items/:id", route.Path)
	assert.Equal(t, "5", p["id"])

	h, _ = r.Find("GET", "/data", nil)
	assert.NotNil(t, h)
}

func TestRouter_NotFoundForWrongMethod(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddStatic("GET", "/secret", dummyHandler, &Route{})

	h, _ := r.Find("POST", "/secret", nil)
	assert.NotNil(t, h)
}

func TestRouter_Reset(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddStatic("GET", "/test", dummyHandler, &Route{Path: "/test", Method: "GET"})

	h, _ := r.Find("GET", "/test", nil)
	assert.NotNil(t, h)

	r.Reset()

	h, _ = r.Find("GET", "/test", nil)
	assert.NotNil(t, h)
}

func TestRouter_GetRouteCount(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddStatic("GET", "/a", dummyHandler, &Route{})
	r.AddStatic("GET", "/b", dummyHandler, &Route{})

	s := r.GetRouteCount()
	assert.Equal(t, 2, s)
}

func TestRouter_StaticFallback(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddStatic("GET", "/api/health", dummyHandler, &Route{Path: "/api/health"})
	r.AddParametric("GET", "/api/:version/status", dummyHandler, &Route{Path: "/api/:version/status"})

	h, r2 := r.Find("GET", "/api/health", nil)
	assert.NotNil(t, h)
	assert.Equal(t, "/api/health", r2.Path)

	p := map[string]string{}
	h, _ = r.Find("GET", "/api/v1/status", p)
	assert.NotNil(t, h)
	assert.Equal(t, "v1", p["version"])
}

func TestRouter_MethodParametricIsolation(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddParametric("GET", "/items/:id", dummyHandler, &Route{Path: "/items/:id", Method: "GET"})
	r.AddParametric("POST", "/items/:id", dummyHandler, &Route{Path: "/items/:id", Method: "POST"})

	p := map[string]string{}
	h, _ := r.Find("GET", "/items/1", p)
	assert.NotNil(t, h)
	assert.Equal(t, "1", p["id"])

	p = map[string]string{}
	h, _ = r.Find("POST", "/items/2", p)
	assert.NotNil(t, h)
	assert.Equal(t, "2", p["id"])
}

func TestRadixTree_MixedSlashLevels(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/api/v1/users", dummyHandler, &Route{Path: "/api/v1/users"})
	tree.insert("GET", "/api/v2/users", dummyHandler, &Route{Path: "/api/v2/users"})

	h, r, ok := tree.search("GET", "/api/v1/users", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/api/v1/users", r.Path)

	h, r, ok = tree.search("GET", "/api/v2/users", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/api/v2/users", r.Path)
}

func TestRadixTree_DeepWildcard(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/files/images/*", dummyHandler, &Route{Path: "/files/images/*"})

	p := map[string]string{}
	h, _, ok := tree.search("GET", "/files/images/2024/photo.jpg", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "2024/photo.jpg", p["*"])

	p = map[string]string{}
	h, _, ok = tree.search("GET", "/files/images/logo.png", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "logo.png", p["*"])
}

func TestRadixTree_PrefixCompression_UserUsers(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/user", dummyHandler, &Route{Path: "/user"})
	tree.insert("GET", "/users", dummyHandler, &Route{Path: "/users"})

	h, r, ok := tree.search("GET", "/user", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/user", r.Path)

	h, r, ok = tree.search("GET", "/users", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users", r.Path)
}

func TestRadixTree_PrefixCompression_UsersProfile(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/users", dummyHandler, &Route{Path: "/users"})
	tree.insert("GET", "/users/profile", dummyHandler, &Route{Path: "/users/profile"})

	h, r, ok := tree.search("GET", "/users", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users", r.Path)

	h, r, ok = tree.search("GET", "/users/profile", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/profile", r.Path)
}

func TestRadixTree_PrefixCompression_Combined(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/user", dummyHandler, &Route{Path: "/user"})
	tree.insert("GET", "/users", dummyHandler, &Route{Path: "/users"})
	tree.insert("GET", "/users/profile", dummyHandler, &Route{Path: "/users/profile"})

	h, r, ok := tree.search("GET", "/user", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/user", r.Path)

	h, r, ok = tree.search("GET", "/users", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users", r.Path)

	h, r, ok = tree.search("GET", "/users/profile", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/users/profile", r.Path)

	h, _, ok = tree.search("GET", "/usersettings", nil)
	assert.False(t, ok)
	assert.Nil(t, h)
}

func TestRadixTree_PrefixCompression_MultipleLevels(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/api/v1/health", dummyHandler, &Route{Path: "/api/v1/health"})
	tree.insert("GET", "/api/v2/health", dummyHandler, &Route{Path: "/api/v2/health"})
	tree.insert("GET", "/api/v1/status", dummyHandler, &Route{Path: "/api/v1/status"})

	h, r, ok := tree.search("GET", "/api/v1/health", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/api/v1/health", r.Path)

	h, r, ok = tree.search("GET", "/api/v2/health", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/api/v2/health", r.Path)

	h, r, ok = tree.search("GET", "/api/v1/status", nil)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/api/v1/status", r.Path)
}

func TestRouter_MethodNotAllowed(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddStatic("GET", "/secret", dummyHandler, &Route{Method: "GET", Path: "/secret"})

	h, r2 := r.Find("POST", "/secret", nil)
	assert.NotNil(t, h)
	assert.Nil(t, r2)
}

func TestRouter_MethodNotAllowed_Param(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddParametric("GET", "/items/:id", dummyHandler, &Route{Method: "GET", Path: "/items/:id"})

	h, _ := r.Find("POST", "/items/42", nil)
	assert.NotNil(t, h)
}

func TestRouter_404_Nonexistent(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddStatic("GET", "/exists", dummyHandler, &Route{Path: "/exists"})

	h, r2 := r.Find("GET", "/nonexistent", nil)
	assert.NotNil(t, h)
	assert.Nil(t, r2)
}

func TestRadixTree_ParamAndWildcard_Coexist(t *testing.T) {
	tree := newRadixTree()
	tree.insert("GET", "/files/:type", dummyHandler, &Route{Path: "/files/:type"})
	tree.insert("GET", "/files/*", dummyHandler, &Route{Path: "/files/*"})

	p := map[string]string{}
	h, r, ok := tree.search("GET", "/files/image", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/files/:type", r.Path)
	assert.Equal(t, "image", p["type"])

	p = map[string]string{}
	h, r, ok = tree.search("GET", "/files/docs/report.pdf", p)
	assert.True(t, ok)
	assert.NotNil(t, h)
	assert.Equal(t, "/files/*", r.Path)
	assert.Equal(t, "docs/report.pdf", p["*"])
}

func TestRouter_ParamAndWildcard_Fallthrough(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddParametric("GET", "/files/:type", dummyHandler, &Route{Path: "/files/:type"})
	r.AddWildcard("GET", "/files/", dummyHandler, &Route{Path: "/files/*"})

	p := map[string]string{}
	h, r2 := r.Find("GET", "/files/image.jpg", p)
	assert.NotNil(t, h)
	assert.Equal(t, "/files/:type", r2.Path)
	assert.Equal(t, "image.jpg", p["type"])

	p = map[string]string{}
	h, r2 = r.Find("GET", "/files/docs/report.pdf", p)
	assert.NotNil(t, h)
	assert.Equal(t, "/files/*", r2.Path)
	assert.Equal(t, "docs/report.pdf", p["*"])
}
