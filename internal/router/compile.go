package router

import (
	"strings"

	"github.com/Dziqha/TurboGo/core"
)

// ===== Compiled-switch routing =====
//
// Compile() transcribes the radix tree node-for-node into per-method
// decision structures, changing only:
//
//   - one tree per method instead of a shared [9] handler array,
//   - static routes are NOT transcribed (exact static matches keep using
//     the O(1) per-method maps, which already bypass the tree),
//   - wildcard leaves go to an ordered prefix list (matched by global
//     emission order against decision matches, reproducing backtracking),
//   - param captures go to stack slots with a single map write on success
//     instead of write+delete on backtrack.
//
// Traversal, fragment edges, backtracking order, slash handling and param
// names are all identical to the tree, so dispatch is exact by
// construction. Anything the table cannot express (unknown methods,
// over-deep paths, static-only hits) falls back to the tree.

// cLeaf is a transcribed parametric route leaf.
type cLeaf struct {
	handler core.Handler
	route   *Route
	ord     int      // global DFS emission order (radix attempt order)
	names   []string // param names in path order (tree nodes, first-registration wins)
	pdepths []int    // decision depths holding each param (parallel to names)
}

type cWildcard struct {
	prefix  string // literal prefix including trailing "/", e.g. "/assets/"
	handler core.Handler
	route   *Route
	ord     int // global DFS emission order (radix attempt order)
}

// Compiled is the flattened dispatch table for one router snapshot.
// The decision tree is shared across methods (leaves are per-method),
// mirroring the radix tree 1:1. Rebuild via Compile() after route changes;
// safe for concurrent reads once published (entries are immutable).
type Compiled struct {
	root *cDecNode      // transcribed radix root (nil when router empty)
	wild [9][]*cWildcard // wildcards per methodIdx (radix attempt order)
}

// cDecNode mirrors one radix tree node: fragment edges in tree order,
// one param edge, and per-method leaves.
type cDecNode struct {
	edges []cDecEdge // fragment edges in tree-indices order (distinct first bytes)
	param *cDecNode  // param edge (nil if none)
	pname string     // tree param node name (first-registration wins)
	leaf  [9]*cLeaf // route ending exactly here per method (nil if none)
}

// cDecEdge is one fragment edge.
type cDecEdge struct {
	frag  string
	child *cDecNode
}

// Compile transcribes the current radix tree into a Compiled table.
// It does not modify the router; callers decide when to swap it in.
func (r *SingleThreadedRouter) Compile() *Compiled {
	c := &Compiled{}
	if r.tree == nil || r.tree.root == nil {
		return c
	}
	st := &compStage{}
	c.root = st.build(r.tree.root, nil, nil, 0)
	c.wild = st.wild
	return c
}

// compStage carries the shared emission counter across param leaves and
// wildcard entries so their relative order matches radix attempt order.
type compStage struct {
	wild [9][]*cWildcard
	seq  int
}

// build transcribes a radix subtree. names/pdepths accumulate the param
// names and decision depths along the current path.
func (st *compStage) build(n *node, names []string, pdepths []int, depth int) *cDecNode {
	d := &cDecNode{}
	for mi, e := range n.handlers {
		if e == nil || e.route == nil {
			continue
		}
		if !isParamPattern(e.route.Path) {
			continue // static: served by the exact-match maps
		}
		d.leaf[mi] = &cLeaf{
			handler: e.handler,
			route:   e.route,
			ord:     st.seq,
			names:   append([]string(nil), names...),
			pdepths: append([]int(nil), pdepths...),
		}
		st.seq++
	}
	for _, ch := range n.children {
		d.edges = append(d.edges, cDecEdge{
			frag:  ch.segment,
			child: st.build(ch, names, pdepths, depth+1),
		})
	}
	if n.paramChild != nil {
		d.param = st.build(n.paramChild,
			append(names, n.paramChild.paramName),
			append(pdepths, depth), depth+1)
		d.pname = n.paramChild.paramName
	}
	if n.wildcardChild != nil {
		for mi, e := range n.wildcardChild.handlers {
			if e == nil || e.route == nil {
				continue
			}
			st.wild[mi] = append(st.wild[mi], &cWildcard{
				prefix:  strings.TrimSuffix(e.route.Path, "*"),
				handler: e.handler,
				route:   e.route,
				ord:     st.seq,
			})
			st.seq++
		}
	}
	return d
}

// isParamPattern reports whether a route pattern holds a ":name" segment.
func isParamPattern(pattern string) bool {
	for i := 0; i+1 < len(pattern); i++ {
		if pattern[i] == '/' && pattern[i+1] == ':' {
			return true
		}
	}
	return false
}

// ParamRouteCount returns the number of compiled parametric routes (tests/metrics).
func (c *Compiled) ParamRouteCount() int {
	n := 0
	var walk func(d *cDecNode)
	walk = func(d *cDecNode) {
		if d == nil {
			return
		}
		for mi := 0; mi < 9; mi++ {
			if d.leaf[mi] != nil {
				n++
			}
		}
		for i := range d.edges {
			walk(d.edges[i].child)
		}
		walk(d.param)
	}
	walk(c.root)
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

// ===== Compiled dispatch (zero-alloc lookup) =====

// methodIndex maps a method to its table slot with a switch (no map
// lookup on the hot path). Unknown methods report false: tree.search
// aliases them to the GET slot, so callers must use the tree for those
// to preserve exact behavior.
func methodIndex(method string) (int, bool) {
	switch method {
	case "GET":
		return 0, true
	case "POST":
		return 1, true
	case "PUT":
		return 2, true
	case "DELETE":
		return 3, true
	case "PATCH":
		return 4, true
	case "HEAD":
		return 5, true
	case "OPTIONS":
		return 6, true
	case "CONNECT":
		return 7, true
	case "TRACE":
		return 8, true
	default:
		return 0, false
	}
}

// maxSegs bounds decision depth before falling back to tree search
// (correctness first; real routes are far shallower).
const maxSegs = 16

// search mirrors node.search exactly: single-slash skip, fragment edge
// compare (first-byte distinct, may span '/'), param capture to next '/'
// into stack slots, wildcard-free (resolved globally by ord). Params are
// never touched here; the caller copies slots on success. overflow is
// true when the path is deeper than maxSegs.
func (n *cDecNode) search(path string, pos, depth, mi int, slots *[maxSegs][2]int) (*cLeaf, bool) {
	if pos < len(path) && path[pos] == '/' {
		pos++
		if pos >= len(path) {
			return n.leaf[mi], false
		}
	} else if pos >= len(path) {
		return n.leaf[mi], false
	}
	if depth >= maxSegs {
		return nil, true
	}
	fb := path[pos]
	for i := range n.edges {
		if n.edges[i].frag[0] != fb {
			continue
		}
		frag := n.edges[i].frag
		if pos+len(frag) > len(path) {
			break // first byte matched but fragment overruns: no other edge can match
		}
		match := true
		for j := 0; j < len(frag); j++ {
			if frag[j] != path[pos+j] {
				match = false
				break
			}
		}
		if match {
			if leaf, ovf := n.edges[i].child.search(path, pos+len(frag), depth+1, mi, slots); ovf || leaf != nil {
				return leaf, ovf
			}
			break
		}
		break // distinct first bytes: at most one edge can match
	}
	if n.param != nil {
		s := pos
		for pos < len(path) && path[pos] != '/' {
			pos++
		}
		if pos > s {
			slots[depth][0], slots[depth][1] = s, pos
			if leaf, ovf := n.param.search(path, pos, depth+1, mi, slots); ovf || leaf != nil {
				return leaf, ovf
			}
		}
	}
	return nil, false
}

// SetCompiled enables or disables the compiled-switch fast path.
// Enabling (re)builds the table from the current routes.
func (r *SingleThreadedRouter) SetCompiled(on bool) {
	if on {
		r.compiled = r.Compile()
	}
	r.compiledOn = on
}
