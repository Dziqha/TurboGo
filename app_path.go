package TurboGo

import "github.com/valyala/fasthttp"

// routePath returns the path used for routing.
//
// fasthttp's ctx.Path() parses the request URI and normalizes the path
// (percent-decoding, duplicate slashes, /./ and /../ resolution). That
// parse dominates handler CPU for trivial routes, yet for the overwhelming
// majority of real request targets it is a no-op. routePath therefore uses
// the raw request target directly whenever normalization provably cannot
// change it, and falls back to ctx.Path() otherwise, so routing sees
// byte-identical input either way.
//
// raw says which branch was taken (false means "fell back"); it exists so
// tests can prove the fast branch is always equivalent.
func routePath(ctx *fasthttp.RequestCtx) (string, bool) {
	raw := ctx.RequestURI()
	if len(raw) == 0 || raw[0] != '/' {
		// authority-form (CONNECT) or absolute-form: let fasthttp parse.
		return b2s(ctx.Path()), false
	}
	// Mirror URI.parse: the path ends at the query or fragment, whichever
	// comes first (a fragment before '?' makes the query irrelevant).
	end := len(raw)
	for i := 0; i < len(raw); i++ {
		if c := raw[i]; c == '?' || c == '#' {
			end = i
			break
		}
	}
	p := raw[:end]
	if needsPathNormalization(p) {
		return b2s(ctx.Path()), false
	}
	return b2s(p), true
}

// needsPathNormalization reports whether fasthttp's normalizePath could
// change p. It is deliberately conservative: every case normalizePath
// handles (percent-decoding, control bytes, duplicate slashes, /./, /..,
// backslash separators on Windows) returns true.
func needsPathNormalization(p []byte) bool {
	var prev byte
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch {
		case c < ' ' || c == 0x7f:
			return true // CTL byte: fasthttp rejects the URI
		case c == '%', c == '\\':
			return true // percent-decoding / Windows separator handling
		case c == '/':
			if prev == '/' {
				return true // duplicate slash
			}
		case c == '.':
			if prev == '/' {
				// "/." at end or "/./" -> current-dir removal
				if i+1 == len(p) || p[i+1] == '/' {
					return true
				}
				// "/.." at end or "/../" -> parent-dir removal
				if p[i+1] == '.' && (i+2 == len(p) || p[i+2] == '/') {
					return true
				}
			}
		}
		prev = c
	}
	return false
}
