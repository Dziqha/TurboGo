package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_Counts(t *testing.T) {
	r := prepareBenchRouter() // 2 static, 3 param, 1 wildcard (GET)
	c := r.Compile()

	assert.Equal(t, 3, c.ParamRouteCount())
	assert.Equal(t, 1, c.WildcardCount())
	require.NotNil(t, c.root)
}

func TestCompile_MirrorsFragments(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddParametric("GET", "/a/:x", dummyHandler, &Route{Path: "/a/:x"})
	r.AddParametric("GET", "/api/:v/users", dummyHandler, &Route{Path: "/api/:v/users"})

	c := r.Compile()
	require.NotNil(t, c.root)
	// Root fragment edge "a" shared by both patterns (prefix split).
	require.Len(t, c.root.edges, 1)
	assert.Equal(t, "a", c.root.edges[0].frag)
}

func TestCompile_ParamNamesFirstRegistrationWins(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddParametric("GET", "/users/:id", dummyHandler, &Route{Path: "/users/:id"})
	r.AddParametric("GET", "/users/:name", dummyHandler, &Route{Path: "/users/:name"})

	c := r.Compile()
	assert.Equal(t, 1, c.ParamRouteCount())
}

func TestCompile_WildcardPrefix(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddWildcard("GET", "/assets/", dummyHandler, &Route{Path: "/assets/*"})

	c := r.Compile()
	require.Len(t, c.wild[methodIdx["GET"]], 1)
	assert.Equal(t, "/assets/", c.wild[methodIdx["GET"]][0].prefix)
}

func TestCompile_Empty(t *testing.T) {
	r := NewSingleThreadedRouter()
	c := r.Compile()
	assert.Equal(t, 0, c.ParamRouteCount())
	assert.Equal(t, 0, c.WildcardCount())
}

func TestCompile_DecisionTreeLookup(t *testing.T) {
	r := prepareBenchRouter()
	r.SetCompiled(true)

	p := map[string]string{}
	h, rt := r.Find("GET", "/users/42", p)
	require.NotNil(t, h)
	require.NotNil(t, rt)
	assert.Equal(t, "/users/:id", rt.Path)
	assert.Equal(t, "42", p["id"])

	p = map[string]string{}
	_, rt = r.Find("GET", "/users/new-x/posts", p)
	require.NotNil(t, rt)
	assert.Equal(t, "/users/:id/posts", rt.Path)
}

func TestCompiled_DirtyRecompile(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddParametric("GET", "/users/:id", dummyHandler, &Route{Path: "/users/:id"})
	r.SetCompiled(true)

	// Register after compile: table rebuilds lazily on next lookup.
	r.AddParametric("GET", "/orgs/:oid", dummyHandler, &Route{Path: "/orgs/:oid"})

	p := map[string]string{}
	_, rt := r.Find("GET", "/orgs/7", p)
	require.NotNil(t, rt)
	assert.Equal(t, "/orgs/:oid", rt.Path)
	assert.Equal(t, "7", p["oid"])

	// Old routes keep working through the rebuilt table.
	p = map[string]string{}
	_, rt = r.Find("GET", "/users/9", p)
	require.NotNil(t, rt)
	assert.Equal(t, "/users/:id", rt.Path)
	assert.Equal(t, "9", p["id"])
}
