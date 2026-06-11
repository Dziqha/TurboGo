package router

import (
	"strings"

	"github.com/Dziqha/TurboGo/core"
)

type nodeType uint8

const (
	nodeStatic   nodeType = 0
	nodeParam    nodeType = 1
	nodeWildcard nodeType = 2
)

var methodIdx = map[string]int{
	"GET":     0,
	"POST":    1,
	"PUT":     2,
	"DELETE":  3,
	"PATCH":   4,
	"HEAD":    5,
	"OPTIONS": 6,
	"CONNECT": 7,
	"TRACE":   8,
}

type routeEntry struct {
	handler core.Handler
	route   *Route
}

type node struct {
	segment       string
	nType         nodeType
	indices       string
	children      []*node
	paramChild    *node
	wildcardChild *node
	paramName     string
	handlers      [9]*routeEntry
}

type radixTree struct {
	root *node
}

func newRadixTree() *radixTree {
	return &radixTree{root: &node{}}
}

func (t *radixTree) insert(method, fullPath string, handler core.Handler, route *Route) {
	if fullPath == "/" || fullPath == "" {
		t.root.handlers[methodIdx[method]] = &routeEntry{handler: handler, route: route}
		return
	}
	clean := strings.Trim(fullPath, "/")
	segs := strings.Split(clean, "/")
	mi := methodIdx[method]
	t.root.insertSegments(segs, mi, handler, route)
}

func (n *node) insertSegments(segs []string, mi int, handler core.Handler, route *Route) {
	if len(segs) == 0 {
		n.handlers[mi] = &routeEntry{handler: handler, route: route}
		return
	}

	seg := segs[0]
	rest := segs[1:]

	if len(seg) > 0 && seg[0] == ':' {
		name := seg[1:]
		if n.paramChild == nil {
			n.paramChild = &node{nType: nodeParam, paramName: name}
		}
		n.paramChild.insertSegments(rest, mi, handler, route)
		return
	}

	if seg == "*" {
		n.wildcardChild = &node{nType: nodeWildcard}
		n.wildcardChild.handlers[mi] = &routeEntry{handler: handler, route: route}
		return
	}

	for _, child := range n.children {
		common := commonPrefixLen(seg, child.segment)
		if common == 0 {
			continue
		}

		if common < len(child.segment) {
			split := &node{
				segment:       child.segment[common:],
				children:      child.children,
				indices:       child.indices,
				paramChild:    child.paramChild,
				wildcardChild: child.wildcardChild,
				handlers:      child.handlers,
				paramName:     child.paramName,
				nType:         child.nType,
			}
			child.segment = child.segment[:common]
			child.children = []*node{split}
			child.indices = string(split.segment[0])
			child.paramChild = nil
			child.wildcardChild = nil
			child.handlers = [9]*routeEntry{}
		}

		remaining := seg[common:]
		if remaining == "" {
			child.insertSegments(rest, mi, handler, route)
		} else {
			child.insertRemaining(remaining, rest, mi, handler, route)
		}
		return
	}

	n.indices += string(seg[0])
	child := &node{segment: seg}
	child.insertSegments(rest, mi, handler, route)
	n.children = append(n.children, child)
}

func (n *node) insertRemaining(remaining string, rest []string, mi int, handler core.Handler, route *Route) {
	for _, child := range n.children {
		common := commonPrefixLen(remaining, child.segment)
		if common == 0 {
			continue
		}

		if common < len(child.segment) {
			split := &node{
				segment:       child.segment[common:],
				children:      child.children,
				indices:       child.indices,
				paramChild:    child.paramChild,
				wildcardChild: child.wildcardChild,
				handlers:      child.handlers,
				paramName:     child.paramName,
				nType:         child.nType,
			}
			child.segment = child.segment[:common]
			child.children = []*node{split}
			child.indices = string(split.segment[0])
			child.paramChild = nil
			child.wildcardChild = nil
			child.handlers = [9]*routeEntry{}
		}

		remaining2 := remaining[common:]
		if remaining2 == "" {
			child.insertSegments(rest, mi, handler, route)
		} else {
			child.insertRemaining(remaining2, rest, mi, handler, route)
		}
		return
	}

	n.indices += string(remaining[0])
	child := &node{segment: remaining}
	child.insertSegments(rest, mi, handler, route)
	n.children = append(n.children, child)
}

func commonPrefixLen(a, b string) int {
	i := 0
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}

func (t *radixTree) search(method, path string, params map[string]string) (core.Handler, *Route, bool) {
	if path == "/" || path == "" {
		entry := t.root.handlers[methodIdx[method]]
		if entry != nil {
			return entry.handler, entry.route, true
		}
		return nil, nil, false
	}

	mi := methodIdx[method]
	return t.root.search(path, 1, mi, params)
}

func (n *node) search(path string, pos int, mi int, params map[string]string) (core.Handler, *Route, bool) {
	if pos >= len(path) {
		if n.handlers[mi] != nil {
			return n.handlers[mi].handler, n.handlers[mi].route, true
		}
		return nil, nil, false
	}

	if path[pos] == '/' {
		pos++
		if pos >= len(path) {
			if n.handlers[mi] != nil {
				return n.handlers[mi].handler, n.handlers[mi].route, true
			}
			return nil, nil, false
		}
	}

	firstByte := path[pos]
	for i := 0; i < len(n.indices); i++ {
		if n.indices[i] == firstByte {
			child := n.children[i]
			seg := child.segment
			if pos+len(seg) <= len(path) {
				match := true
				for j := 0; j < len(seg); j++ {
					if seg[j] != path[pos+j] {
						match = false
						break
					}
				}
				if match {
					h, r, ok := child.search(path, pos+len(seg), mi, params)
					if ok {
						return h, r, ok
					}
				}
			}
			break
		}
	}

	if n.paramChild != nil {
		savedPos := pos
		end := pos
		for end < len(path) && path[end] != '/' {
			end++
		}
		if end > pos {
			if params != nil {
				params[n.paramChild.paramName] = path[pos:end]
			}
			h, r, ok := n.paramChild.search(path, end, mi, params)
			if ok {
				return h, r, ok
			}
			if params != nil {
				delete(params, n.paramChild.paramName)
			}
		}
		pos = savedPos
	}

	if n.wildcardChild != nil && n.wildcardChild.handlers[mi] != nil {
		if pos < len(path) && path[pos] == '/' {
			pos++
		}
		if params != nil {
			params["*"] = path[pos:]
		}
		return n.wildcardChild.handlers[mi].handler, n.wildcardChild.handlers[mi].route, true
	}

	return nil, nil, false
}

func (n *node) hasAnyHandler() bool {
	for _, h := range n.handlers {
		if h != nil {
			return true
		}
	}
	return false
}

func (t *radixTree) pathExistsAny(path string) bool {
	if path == "/" || path == "" {
		return t.root.hasAnyHandler()
	}
	for mi := 0; mi < 9; mi++ {
		_, _, ok := t.root.search(path, 1, mi, nil)
		if ok {
			return true
		}
	}
	return false
}
