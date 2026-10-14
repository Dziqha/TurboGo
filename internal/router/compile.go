package router

import (
	"strings"

	"github.com/Dziqha/TurboGo/core"
)

// ===== Compiled-switch routing =====
//
// Compile() walks the radix tree in exact search order (static children in
// indices order, then param child, then wildcard) and flattens parametric
// and wildcard routes into ordered dispatch tables:
//
//   - param routes grouped by (method, segment count) so lookup starts with
//     a segment-count switch instead of a tree walk,
//   - wildcards ordered deepest-prefix-first, mirroring radix backtracking,
//   - static routes are NOT emitted: exact static matches keep using the
//     O(1) per-method maps, which already bypass the tree.
//
// Because emission order equals radix attempt order (filtered to the same
// segment count, which can never change the winner), compiled dispatch
// returns byte-identical results to tree search, including param names
// (first-registration wins, as the tree reuses param nodes).

type cParamRoute struct {
	lits    []string // per-segment literal; "" where the segment is a param
	isParam []bool
	names   []string // effective param names in order (tree nodes, first-registration wins)
	handler core.Handler
	route   *Route
}

type cWildcard struct {
	prefix  string // literal prefix including trailing "/", e.g. "/assets/"
	handler core.Handler
	route   *Route
}

// Compiled is the flattened dispatch table for one router snapshot.
// Rebuild via Compile() after route changes; safe for concurrent reads
// once published (entries are immutable).
type Compiled struct {
	params map[int]map[int][]*cParamRoute // methodIdx -> segCount -> routes (radix attempt order)
	wild   map[int][]*cWildcard            // methodIdx -> wildcards (deepest first)
}

// Compile flattens the current radix tree into a Compiled table.
// It does not modify the router; callers decide when to swap it in.
func (r *SingleThreadedRouter) Compile() *Compiled {
	c := &Compiled{
		params: make(map[int]map[int][]*cParamRoute),
		wild:   make(map[int][]*cWildcard),
	}
	if r.tree == nil || r.tree.root == nil {
		return c
	}
	r.tree.root.emit(c, nil)
	return c
}

// segCountOf returns the number of "/" separated segments of a path
// using the same normalization as tree insert (trim leading/trailing "/").
func segCountOf(path string) int {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return 0
	}
	return strings.Count(clean, "/") + 1
}

// splitSegs splits a pattern the same way tree insert does.
func splitSegs(pattern string) []string {
	clean := strings.Trim(pattern, "/")
	if clean == "" {
		return nil
	}
	return strings.Split(clean, "/")
}

// emit walks the subtree in exact search order and appends leaves.
func (n *node) emit(c *Compiled, pnames []string) {
	// Node handlers: param-skeleton leaves are emitted, static-skeleton
	// leaves are skipped (served by the static exact-match maps).
	for mi, e := range n.handlers {
		if e == nil || e.route == nil {
			continue
		}
		segs := splitSegs(e.route.Path)
		hasParam := false
		for _, s := range segs {
			if len(s) > 0 && s[0] == ':' {
				hasParam = true
				break
			}
		}
		if !hasParam {
			continue
		}
		pr := &cParamRoute{
			lits:    make([]string, len(segs)),
			isParam: make([]bool, len(segs)),
			names:   append([]string(nil), pnames...),
			handler: e.handler,
			route:   e.route,
		}
		for i, s := range segs {
			if len(s) > 0 && s[0] == ':' {
				pr.isParam[i] = true
			} else {
				pr.lits[i] = s
			}
		}
		if c.params[mi] == nil {
			c.params[mi] = make(map[int][]*cParamRoute)
		}
		c.params[mi][len(segs)] = append(c.params[mi][len(segs)], pr)
	}

	// Static children in indices order (mirrors search loop).
	for _, child := range n.children {
		child.emit(c, pnames)
	}

	// Param child: extend effective names with this node's name.
	if n.paramChild != nil {
		n.paramChild.emit(c, append(pnames, n.paramChild.paramName))
	}

	// Wildcard child: prefix is the registered pattern minus trailing "*".
	if n.wildcardChild != nil {
		for mi, e := range n.wildcardChild.handlers {
			if e == nil || e.route == nil {
				continue
			}
			prefix := strings.TrimSuffix(e.route.Path, "*")
			c.wild[mi] = append(c.wild[mi], &cWildcard{
				prefix:  prefix,
				handler: e.handler,
				route:   e.route,
			})
		}
	}
}

// ParamRouteCount returns the number of compiled parametric routes (tests/metrics).
func (c *Compiled) ParamRouteCount() int {
	n := 0
	for _, bySeg := range c.params {
		for _, list := range bySeg {
			n += len(list)
		}
	}
	return n
}

// WildcardCount returns the number of compiled wildcard routes.
func (c *Compiled) WildcardCount() int {
	n := 0
	for _, list := range c.wild {
		n += len(list)
	}
	return n
}
