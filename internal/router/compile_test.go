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

	// Grouped by segment count: /users/:id (2), /api/:version/users (3), /users/:id/posts (3)
	bySeg := c.params[methodIdx["GET"]]
	require.NotNil(t, bySeg)
	assert.Len(t, bySeg[2], 1)
	assert.Len(t, bySeg[3], 2)
}

func TestCompile_ParamNamesFirstRegistrationWins(t *testing.T) {
	r := NewSingleThreadedRouter()
	r.AddParametric("GET", "/users/:id", dummyHandler, &Route{Path: "/users/:id"})
	r.AddParametric("GET", "/users/:name", dummyHandler, &Route{Path: "/users/:name"})

	c := r.Compile()
	list := c.params[methodIdx["GET"]][2]
	// Same skeleton overwrites the same tree leaf, so only one entry
	// remains, keeping the first-registered param name.
	require.Len(t, list, 1)
	assert.Equal(t, []string{"id"}, list[0].names)
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
