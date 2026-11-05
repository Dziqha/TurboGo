package TurboGo

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func ctxForTarget(target string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&fasthttp.Request{}, nil, nil)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI(target)
	return ctx
}

// TestRoutePath_Equivalence is the safety net for the fast branch: for any
// target where routePath skips parsing, the result must be byte-identical
// to what fasthttp's parser+normalizer would produce.
func TestRoutePath_Equivalence(t *testing.T) {
	seeds := []string{
		"/", "/plaintext", "/user/42", "/users/", "/a/b/c", "/.hidden",
		"/a.../b", "/a/b.", "/v1.2/x", "/a*b", "/trailing/",
		// normalization cases: must fall back and still match
		"", "*", "http://example.com/x", "//", "///", "/a//b", "/a/./b",
		"/a/../b", "/a/.", "/a/..", "/.", "/..", "/a/.b/c", "/a/..b/c",
		"/a%2Fb", "/a%20b", "/a\\b", "/a?x=1", "/a#frag", "/a#f?x=1",
		"/a?x=1#f", "/a/b?q=//x", "/a/b#f//x", "/x%", "//a", "/ a", "/a\x01b",
	}
	rng := rand.New(rand.NewSource(11))
	alpha := []byte("ab/./%?#\\ ~_-.x:0123")
	for i := 0; i < 50000; i++ {
		n := rng.Intn(12)
		var sb strings.Builder
		if rng.Intn(4) > 0 {
			sb.WriteByte('/')
		}
		for k := 0; k < n; k++ {
			sb.WriteByte(alpha[rng.Intn(len(alpha))])
		}
		seeds = append(seeds, sb.String())
	}

	fast, slow := 0, 0
	for _, target := range seeds {
		ctx := ctxForTarget(target)
		got, raw := routePath(ctx)
		want := string(ctx.Path())
		if raw {
			fast++
			if got != want {
				t.Fatalf("fast branch %q routed %q, parser says %q", target, got, want)
			}
		} else {
			slow++
			if got != want {
				t.Fatalf("fallback %q routed %q, parser says %q", target, got, want)
			}
		}
	}
	t.Logf("targets=%d fast=%d fallback=%d", len(seeds), fast, slow)
	if fast == 0 {
		t.Fatal("fast branch never taken; guard is too conservative")
	}
}

func TestRoutePath_FastBranchCoversNormalTraffic(t *testing.T) {
	for _, target := range []string{
		"/", "/plaintext", "/json", "/user/42", "/users/42/posts/7",
		"/api/v1/items/abc", "/assets/images/logo.png", "/health",
	} {
		ctx := ctxForTarget(target)
		got, raw := routePath(ctx)
		if !raw {
			t.Errorf("%q should use the fast branch", target)
			continue
		}
		if want := string(ctx.Path()); got != want {
			t.Errorf("%q: got %q want %q", target, got, want)
		}
	}
}

func TestRoutePath_QueryAndFragmentTrimming(t *testing.T) {
	cases := map[string]string{
		"/user/42?queries=20": "/user/42",
		"/user/42#frag":       "/user/42",
		"/user/42?a=1#f":      "/user/42",
		"/user/42#f?a=1":      "/user/42",
	}
	for target, want := range cases {
		got, raw := routePath(ctxForTarget(target))
		if !raw {
			t.Errorf("%q fell back unexpectedly", target)
			continue
		}
		if got != want {
			t.Errorf("%q -> %q, want %q", target, got, want)
		}
	}
}

func BenchmarkRoutePath(b *testing.B) {
	ctx := ctxForTarget("/user/42")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = routePath(ctx)
	}
}

func BenchmarkCtxPath(b *testing.B) {
	ctx := ctxForTarget("/user/42")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx.Request.Reset()
		ctx.Request.SetRequestURI("/user/42")
		_ = ctx.Path()
	}
}
