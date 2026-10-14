package router

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/Dziqha/TurboGo/core"
	"github.com/stretchr/testify/require"
)

// handler stem cells: distinct funcs so identity is comparable.
func mkH(tag string) core.Handler {
	return func(c *core.Context) { _ = tag }
}

func hptr(h core.Handler) uintptr {
	if h == nil {
		return 0
	}
	return reflect.ValueOf(h).Pointer()
}

func buildFuzzRouter() *SingleThreadedRouter {
	r := NewSingleThreadedRouter()
	add := func(method, pattern string) {
		h := mkH(method + " " + pattern)
		rt := &Route{Path: pattern, Method: method}
		switch {
		case pattern == "/" || (!hasParam(pattern) && !hasWild(pattern)):
			r.AddStatic(method, pattern, h, rt)
		case hasWild(pattern):
			r.AddWildcard(method, pattern[:len(pattern)-1], h, rt)
		default:
			r.AddParametric(method, pattern, h, rt)
		}
	}
	// Static / param conflicts, shared edges, multi-method, wildcards.
	for _, m := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"} {
		add(m, "/")
		add(m, "/users/new")
		add(m, "/users/:id")
		add(m, "/users/:id/posts")
		add(m, "/users/:id/posts/:pid")
		add(m, "/:a/:b")
		add(m, "/a/:x")
		add(m, "/a/:y/b")
		add(m, "/api/:version/users")
		add(m, "/api/v1/special")
		add(m, "/deep/nest/level/four/:p/five")
	}
	// Method-specific: only POST/PUT here (drives 405 for GET).
	add("POST", "/only-post/:id")
	add("PUT", "/only-post/:id")
	// Wildcards at multiple depths.
	add("GET", "/assets/*")
	add("GET", "/assets/images/*")
	add("POST", "/upload/*")
	// Same skeleton, different names (first-registration-wins names).
	add("GET", "/dup/:id")
	add("GET", "/dup/:name")
	return r
}

func hasParam(p string) bool {
	for i := 0; i < len(p); i++ {
		if p[i] == ':' {
			return true
		}
	}
	return false
}

func hasWild(p string) bool {
	return len(p) > 0 && p[len(p)-1] == '*'
}

type fuzzResult struct {
	h    uintptr
	path string // route path, "" when 404/405
	par  map[string]string
}

func runFind(r *SingleThreadedRouter, method, path string) fuzzResult {
	p := map[string]string{}
	h, rt := r.Find(method, path, p)
	fr := fuzzResult{h: hptr(h), par: p}
	if rt != nil {
		fr.path = rt.Path
	}
	return fr
}

func TestFuzz_CompiledMatchesTree(t *testing.T) {
	tree := buildFuzzRouter()
	comp := buildFuzzRouter()
	comp.SetCompiled(true)

	seeds := []string{"", "/", "42", "new", "x", "v1", "v2", "a", "b", "c",
		"hello-world", "a.b", "x_y", "007", "UPPER", "with space", "%20",
		"logo.png", "a/b", "x/y/z", "../escape", "trailing/", ".", "..",
		"verylongsegment" + string(make([]byte, 0))}
	longSeg := ""
	for i := 0; i < 300; i++ {
		longSeg += "z"
	}
	seeds = append(seeds, longSeg)

	bases := []string{"", "/", "/users", "/users/", "/api", "/api/v1",
		"/api/v2", "/assets", "/assets/", "/assets/images", "/dup", "/only-post",
		"/deep/nest/level/four/X", "/a", "/missing", "/u", "/users/new/extra"}

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE", "PURGE"}

	// 1. Hand-picked edge cases.
	paths := []string{
		"/", "//", "///", "/users/", "/users//42", "/users/42/",
		"/assets/", "/assets", "/assets/images/", "/assets/images",
		"/dup/1", "/only-post/9", "/a/b", "/a/b/", "/x/y", "/a",
		"/deep/nest/level/four/X/five", "/deep/nest/level/four/X/five/",
		"/api/v1/users", "/api/v9/users", "/api/v1/special",
	}
	for _, b := range bases {
		for _, s := range seeds {
			paths = append(paths, b+"/"+s)
		}
	}

	// 2. Deterministic pseudo-random paths.
	rng := rand.New(rand.NewSource(0xC0FFEE))
	alpha := []byte("ab/:*-._~% ")
	for i := 0; i < 20000; i++ {
		nseg := rng.Intn(5)
		p := ""
		for s := 0; s < nseg; s++ {
			p += "/"
			l := rng.Intn(8)
			for k := 0; k < l; k++ {
				p += string(alpha[rng.Intn(len(alpha))])
			}
		}
		if p == "" {
			p = "/"
		}
		paths = append(paths, p)
	}

	mismatch := 0
	for _, m := range methods {
		for _, p := range paths {
			a := runFind(tree, m, p)
			b := runFind(comp, m, p)
			if a.h != b.h || a.path != b.path || !reflect.DeepEqual(a.par, b.par) {
				mismatch++
				if mismatch <= 5 {
					t.Logf("MISMATCH %s %q: tree=(h=%x,route=%q,par=%v) compiled=(h=%x,route=%q,par=%v)",
						m, p, a.h, a.path, a.par, b.h, b.path, b.par)
				}
			}
		}
	}
	require.Equal(t, 0, mismatch, "compiled dispatch must match radix tree exactly")
}

func TestFuzz_CompiledNilParams(t *testing.T) {
	tree := buildFuzzRouter()
	comp := buildFuzzRouter()
	comp.SetCompiled(true)

	paths := []string{"/", "/users/42", "/users/new", "/a/b", "/x/y",
		"/assets/a/b/c", "/only-post/9", "/missing", "/api/v1/users"}
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "PURGE"}

	for _, m := range methods {
		for _, p := range paths {
			ha, ra := tree.Find(m, p, nil)
			hb, rb := comp.Find(m, p, nil)
			pa, pb := "", ""
			if ra != nil {
				pa = ra.Path
			}
			if rb != nil {
				pb = rb.Path
			}
			require.Equal(t, pa, pb, "route mismatch %s %s", m, p)
			require.Equal(t, hptr(ha), hptr(hb), "handler mismatch %s %s", m, p)
		}
	}
}

func TestFuzz_DeepPathOverflow(t *testing.T) {
	tree := NewSingleThreadedRouter()
	comp := NewSingleThreadedRouter()
	deep := ""
	for i := 0; i < 40; i++ {
		deep += fmt.Sprintf("/s%d", i)
	}
	for _, r := range []*SingleThreadedRouter{tree, comp} {
		r.AddParametric("GET", deep+"/:id", dummyHandler, &Route{Path: deep + "/:id"})
	}
	comp.SetCompiled(true)

	p := map[string]string{}
	ha, ra := tree.Find("GET", deep+"/42", p)
	require.NotNil(t, ha)
	require.NotNil(t, ra)

	p2 := map[string]string{}
	hb, rb := comp.Find("GET", deep+"/42", p2)
	require.Equal(t, hptr(ha), hptr(hb))
	require.Equal(t, ra.Path, rb.Path)
	require.Equal(t, p, p2)
}

// runFindInto mirrors runFind through the zero-alloc slot API.
func runFindInto(r *SingleThreadedRouter, method, path string) fuzzResult {
	var dst [core.MaxSlots]core.ParamSlot
	ovf := map[string]string{}
	h, rt, n := r.FindInto(method, path, &dst, ovf)
	par := map[string]string{}
	for i := 0; i < n; i++ {
		par[dst[i].K] = dst[i].V
	}
	for k, v := range ovf {
		if _, ok := par[k]; !ok {
			par[k] = v
		}
	}
	fr := fuzzResult{h: hptr(h), par: par}
	if rt != nil {
		fr.path = rt.Path
	}
	return fr
}

func TestFuzz_FindIntoMatchesFind(t *testing.T) {
	for _, compiled := range []bool{false, true} {
		r1 := buildFuzzRouter()
		r2 := buildFuzzRouter()
		if compiled {
			r2.SetCompiled(true)
		}
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE", "PURGE"}
		paths := []string{
			"/", "//", "/users/", "/users/42", "/users//42", "/users/42/",
			"/users/new", "/users/1/posts/2", "/a/b", "/x/y", "/a",
			"/assets/", "/assets", "/assets/images/logo.png",
			"/dup/1", "/only-post/9", "/missing", "/api/v1/users",
			"/api/v9/users", "/api/v1/special", "/deep/nest/level/four/X/five",
		}
		rng := rand.New(rand.NewSource(7))
		alpha := []byte("ab/:*-._~% ")
		for i := 0; i < 20000; i++ {
			nseg := rng.Intn(5)
			p := ""
			for s := 0; s < nseg; s++ {
				p += "/"
				l := rng.Intn(8)
				for k := 0; k < l; k++ {
					p += string(alpha[rng.Intn(len(alpha))])
				}
			}
			if p == "" {
				p = "/"
			}
			paths = append(paths, p)
		}
		mismatch := 0
		for _, m := range methods {
			for _, p := range paths {
				a := runFind(r1, m, p)
				b := runFindInto(r2, m, p)
				if a.h != b.h || a.path != b.path || !reflect.DeepEqual(a.par, b.par) {
					mismatch++
					if mismatch <= 5 {
						t.Logf("MISMATCH compiled=%v %s %q: map=(h=%x,route=%q,par=%v) slots=(h=%x,route=%q,par=%v)",
							compiled, m, p, a.h, a.path, a.par, b.h, b.path, b.par)
					}
				}
			}
		}
		require.Equal(t, 0, mismatch, "FindInto must match Find (compiled=%v)", compiled)
	}
}

func TestFuzz_FindIntoOverflow(t *testing.T) {
	r := NewSingleThreadedRouter()
	pat := ""
	for i := 0; i < 10; i++ {
		pat += fmt.Sprintf("/p%d/:v%d", i, i)
	}
	r.AddParametric("GET", pat, dummyHandler, &Route{Path: pat})
	r.SetCompiled(true)

	sample := ""
	for i := 0; i < 10; i++ {
		sample += fmt.Sprintf("/p%d/x%d", i, i)
	}

	var dst [core.MaxSlots]core.ParamSlot
	ovf := map[string]string{}
	_, rt, n := r.FindInto("GET", sample, &dst, ovf)
	require.NotNil(t, rt)
	require.Equal(t, pat, rt.Path)
	// All 10 params must be visible across slots + overflow map.
	got := map[string]string{}
	for i := 0; i < n; i++ {
		got[dst[i].K] = dst[i].V
	}
	for k, v := range ovf {
		if _, ok := got[k]; !ok {
			got[k] = v
		}
	}
	require.Len(t, got, 10)
	for i := 0; i < 10; i++ {
		require.Equal(t, fmt.Sprintf("x%d", i), got[fmt.Sprintf("v%d", i)])
	}
}
