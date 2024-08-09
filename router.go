package goblaze

import (
	"sort"
	"strings"
	"sync"

	"github.com/valyala/fasthttp"
)

var handlerCache sync.Map

type Router struct {
	noCopy No // nolint:structcheck,unused

	// pool *sync.Pool

	trees    map[string]*node
	Handler  func(*Ctx)
	Handlers []Handler
	Method   string
	Name     string
	Path     string

	RedirectTrailingSlash bool

	RedirectFixedPath bool

	HandleMethodNotAllowed bool

	HandleOPTIONS bool

	NotFound fasthttp.RequestHandler

	MethodNotAllowed fasthttp.RequestHandler

	PanicHandler func(*fasthttp.RequestCtx, interface{})
}

type node struct {
	path      string
	wildChild bool
	nType     nodeType
	indices   string
	children  []*node
	handle    Handler
	priority  int
	maxParams uint8
	pool      sync.Pool
}

type nodeType uint8

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func countParams(path string) (n uint8) {
	for i := 0; i < len(path); i++ {
		if path[i] == ':' || path[i] == '*' {
			n++
		}
	}
	return
}

func NewRouter() *Router {
	return &Router{

		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleMethodNotAllowed: true,
		HandleOPTIONS:          true,
		trees:                  make(map[string]*node),
	}
}
func (r *Router) ServeHTTP(ctx *fasthttp.RequestCtx) {

	customCtx := AcquireRequestCtx(ctx) // nolint:ifshort
	defer ReleaseRequestCtx(customCtx)

	handler := r.handleRequest(customCtx)
	if handler != nil {

		handler(customCtx)
	} else {

		ctx.Error("Not Found", fasthttp.StatusNotFound)
	}
}

// handleRequest finds the handler for the request.
func (r *Router) handleRequest(ctx *Ctx) Handler {
	method := String(ctx.Method())
	path := String(ctx.Path())

	if cachedHandler, ok := handlerCache.Load(method + path); ok {
		return cachedHandler.(Handler)
	}

	root := r.trees[method]
	if root == nil {
		// If no handler is found for the method, handle the case properly
		if r.HandleMethodNotAllowed {
			allow := r.allowed(path, method)
			if len(allow) > 0 {
				ctx.response.Header.Set("Allow", allow)
				if r.MethodNotAllowed != nil {
					r.MethodNotAllowed(ctx.RequestCtx)
				} else {
					ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
					ctx.SetContentType("text/plain")
					ctx.SetBodyString(fasthttp.StatusMessage(fasthttp.StatusMethodNotAllowed))
				}
				return nil
			}
		}
		// If still no handler is found, return 404 Not Found
		ctx.Error("Method not allowed", fasthttp.StatusNotFound)
		return nil
	}

	handle, tsr := root.getValue(path)
	if handle != nil {
		handlerCache.Store(method+path, handle)
		return handle
	}

	if method != "CONNECT" && path != "/" {
		code := 301 // Permanent redirect
		if method != "GET" {
			// Temporary redirect for methods other than GET
			code = 307
		}

		if tsr && r.RedirectTrailingSlash {
			var uri string
			if len(path) > 1 && path[len(path)-1] == '/' {
				uri = path[:len(path)-1]
			} else {
				uri = path + "/"
			}

			if len(ctx.QueryArgs().QueryString()) > 0 {
				uri += "?" + string(ctx.QueryArgs().QueryString())
			}

			ctx.Redirect(uri, code)
			return nil
		}

		// Try to fix the request path
		if r.RedirectFixedPath {
			fixedPath := CleanPath(path)
			if fixedPath != path {
				if len(ctx.QueryArgs().QueryString()) > 0 {
					fixedPath += "?" + string(ctx.QueryArgs().QueryString())
				}
				ctx.Redirect(fixedPath, code)
				return nil
			}
		}
	}

	// Handle OPTIONS requests
	if method == "OPTIONS" {
		if r.HandleOPTIONS {
			allow := r.allowed(path, method)
			if len(allow) > 0 {
				ctx.response.Header.Set("Allow", allow)
				return nil
			}
		}
	}

	// Handle 405 Method Not Allowed
	if r.HandleMethodNotAllowed {
		allow := r.allowed(path, method)
		if len(allow) > 0 {
			ctx.response.Header.Set("Allow", allow)
			if r.MethodNotAllowed != nil {
				r.MethodNotAllowed(ctx.RequestCtx)
			} else {
				ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
				ctx.SetContentType("text/plain")
				ctx.SetBodyString(fasthttp.StatusMessage(fasthttp.StatusMethodNotAllowed))
			}
			return nil
		}
	}

	return nil
}

func (r *Router) allowed(path, reqMethod string) (allow string) {
	allowed := make([]string, 0, 10) // Increase capacity to avoid frequent reallocations

	if path == "*" || path == "/*" { // Server-wide
		for method := range r.trees {
			if method == "OPTIONS" {
				continue
			}
			allowed = append(allowed, method)
		}
	} else { // Specific path
		for method := range r.trees {
			if method == reqMethod || method == "OPTIONS" {
				continue
			}
			handle, _ := r.trees[method].getValue(path)
			if handle != nil {
				allowed = append(allowed, method)
			}
		}
	}

	// Add OPTIONS method to the allowed list
	if len(allowed) > 0 {
		allowed = append(allowed, "OPTIONS")
	}

	// Sort methods
	sort.Strings(allowed)

	// Use strings.Builder for efficient concatenation
	var builder strings.Builder
	for i, method := range allowed {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(method)
	}

	return builder.String()
}

func (r *Router) Handle(method, path string, handle Handler) {
	if path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}

	if r.trees == nil {
		r.trees = make(map[string]*node)
	}

	root := r.trees[method]
	if root == nil {
		root = new(node)
		r.trees[method] = root
	}

	root.addRoute(path, handle)
}

// HTTP method shortcuts.
func (r *Router) GET(path string, handle Handler)     { r.Handle("GET", path, handle) }
func (r *Router) HEAD(path string, handle Handler)    { r.Handle("HEAD", path, handle) }
func (r *Router) OPTIONS(path string, handle Handler) { r.Handle("OPTIONS", path, handle) }
func (r *Router) POST(path string, handle Handler)    { r.Handle("POST", path, handle) }
func (r *Router) PUT(path string, handle Handler)     { r.Handle("PUT", path, handle) }
func (r *Router) PATCH(path string, handle Handler)   { r.Handle("PATCH", path, handle) }
func (r *Router) DELETE(path string, handle Handler)  { r.Handle("DELETE", path, handle) }

func (n *node) addRoute(path string, handle Handler) {
	fullPath := path
	n.priority++
	numParams := countParams(path)

	// non-empty tree
	if len(n.path) > 0 || len(n.children) > 0 {
	walk:
		for {
			// Update maxParams of the current node
			if numParams > n.maxParams {
				n.maxParams = numParams
			}

			// Find the longest common prefix.
			i := 0
			max := min(len(path), len(n.path))
			for i < max && path[i] == n.path[i] {
				i++
			}

			// Split edge
			if i < len(n.path) {
				child := node{
					path:      n.path[i:],
					wildChild: n.wildChild,
					nType:     static,
					indices:   n.indices,
					children:  n.children,
					handle:    n.handle,
					priority:  n.priority - 1,
				}

				// Update maxParams (max of all children)
				for i := range child.children {
					if child.children[i].maxParams > child.maxParams {
						child.maxParams = child.children[i].maxParams
					}
				}

				n.children = []*node{&child}
				n.indices = string([]byte{n.path[i]})
				n.path = path[:i]
				n.handle = nil
				n.wildChild = false
			}

			// Make new node a child of this node
			if i < len(path) {
				path = path[i:]

				if n.wildChild {
					n = n.children[0]
					n.priority++

					if numParams > n.maxParams {
						n.maxParams = numParams
					}
					numParams--

					if len(path) >= len(n.path) && n.path == path[:len(n.path)] {
						if len(n.path) >= len(path) || path[len(n.path)] == '/' {
							continue walk
						}
					}

					panic("path segment '" + path +
						"' conflicts with existing wildcard '" + n.path +
						"' in path '" + fullPath + "'")
				}

				c := path[0]

				if n.nType == param && c == '/' && len(n.children) == 1 {
					n = n.children[0]
					n.priority++
					continue walk
				}

				for i := 0; i < len(n.indices); i++ {
					if c == n.indices[i] {
						i = n.incrementChildPrio(i)
						n = n.children[i]
						continue walk
					}
				}

				if c != ':' && c != '*' {
					n.indices += string([]byte{c})
					child := &node{
						maxParams: numParams,
					}
					n.children = append(n.children, child)
					n.incrementChildPrio(len(n.indices) - 1)
					n = child
				}
				n.insertChild(numParams, path, fullPath, handle)
				return

			} else if i == len(path) {
				if n.handle != nil {
					panic("a handle is already registered for path '" + fullPath + "'")
				}
				n.handle = handle
			}
			return
		}
	} else {
		n.insertChild(numParams, path, fullPath, handle)
		n.nType = root
	}
}

func (n *node) insertChild(numParams uint8, path, fullPath string, handle Handler) {
	var offset int

	for i, max := 0, len(path); numParams > 0; i++ {
		c := path[i]
		if c != ':' && c != '*' {
			continue
		}

		end := i + 1
		for end < max && path[end] != '/' {
			switch path[end] {
			case ':', '*':
				panic("only one wildcard per path segment is allowed, has: '" +
					path[i:] + "' in path '" + fullPath + "'")
			default:
				end++
			}
		}

		if len(n.children) > 0 {
			panic("wildcard route '" + path[i:end] +
				"' conflicts with existing children in path '" + fullPath + "'")
		}

		if end-i < 2 {
			panic("wildcards must be named with a non-empty name in path '" + fullPath + "'")
		}

		if c == ':' {
			if i > 0 {
				n.path = path[offset:i]
				offset = i
			}

			child := &node{
				nType:     param,
				maxParams: numParams,
			}
			n.children = []*node{child}
			n.wildChild = true
			n = child
			n.priority++
			numParams--

			if end < max {
				n.path = path[offset:end]
				offset = end

				child := &node{
					maxParams: numParams,
					priority:  1,
				}
				n.children = []*node{child}
				n = child
			}

		} else {
			if end != max || numParams > 1 {
				panic("catch-all routes are only allowed at the end of the path in path '" + fullPath + "'")
			}

			if len(n.path) > 0 && n.path[len(n.path)-1] == '/' {
				panic("catch-all conflicts with existing handle for the path segment root in path '" + fullPath + "'")
			}

			i--
			if path[i] != '/' {
				panic("no / before catch-all in path '" + fullPath + "'")
			}

			n.path = path[offset:i]

			child := &node{
				wildChild: true,
				nType:     catchAll,
				maxParams: 1,
			}
			n.children = []*node{child}
			n.indices = string(path[i])
			n = child
			n.priority++

			child = &node{
				path:      path[i:],
				nType:     catchAll,
				maxParams: 1,
				handle:    handle,
				priority:  1,
			}
			n.children = []*node{child}

			return
		}
	}

	n.path = path[offset:]
	n.handle = handle
}

func (n *node) incrementChildPrio(pos int) int {
	n.children[pos].priority++
	prio := n.children[pos].priority

	// Adjust position (move to front)
	newPos := pos
	for newPos > 0 && n.children[newPos-1].priority < prio {
		// Swap node positions
		n.children[newPos-1], n.children[newPos] = n.children[newPos], n.children[newPos-1]
		newPos--
	}

	// Update indices string
	if newPos != pos {
		n.indices = n.indices[:newPos] + string(n.indices[pos]) + n.indices[newPos:pos] + n.indices[pos+1:]
	}

	return newPos
}

func (n *node) getValue(path string) (handle Handler, tsr bool) {
walk:
	for {
		if len(path) > len(n.path) {
			if path[:len(n.path)] == n.path {
				path = path[len(n.path):]
				if !n.wildChild {
					c := path[0]
					for i := 0; i < len(n.indices); i++ {
						if c == n.indices[i] {
							n = n.children[i]
							continue walk
						}
					}

					tsr = (path == "/" && n.handle != nil)
					return
				}

				n = n.children[0]
				switch n.nType {
				case param:
					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					if end < len(path) {
						if len(n.children) > 0 {
							path = path[end:]
							n = n.children[0]
							continue walk
						}

						tsr = (len(path) == end+1)
						return
					}

					if handle = n.handle; handle != nil {
						return
					} else if len(n.children) == 1 {
						n = n.children[0]
						tsr = (n.path == "/" && n.handle != nil)
					}
					return

				case catchAll:
					handle = n.handle
					return

				default:
					panic("invalid node type")
				}
			}
		} else if path == n.path {
			if handle = n.handle; handle != nil {
				return
			}

			if path == "/" && n.wildChild && n.nType != root {
				tsr = true
				return
			}

			for i := 0; i < len(n.indices); i++ {
				if n.indices[i] == '/' {
					n = n.children[i]
					tsr = (len(n.path) == 1 && n.handle != nil) ||
						(n.nType == catchAll && n.children[0].handle != nil)
					return
				}
			}
			return
		}

		tsr = (path == "/" || len(path)+1 == len(n.path) && n.path[len(path)] == '/' &&
			path == n.path[:len(path)] && n.handle != nil)
		return
	}
}
func CleanPath(p string) string {
	// Turn empty string into "/"
	if p == "" {
		return "/"
	}

	n := len(p)
	var buf []byte

	// Invariants:
	//      reading from path; r is index of next byte to process.
	//      writing to buf; w is index of next byte to write.

	// path must start with '/'
	r := 1
	w := 1

	if p[0] != '/' {
		r = 0
		buf = make([]byte, n+1)
		buf[0] = '/'
	}

	trailing := n > 2 && p[n-1] == '/'

	// A bit more clunky without a 'lazybuf' like the path package, but the loop
	// gets completely inlined (bufApp). So in contrast to the path package this
	// loop has no expensive function calls (except 1x make)

	for r < n {
		switch {
		case p[r] == '/':
			// empty path element, trailing slash is added after the end
			r++

		case p[r] == '.' && r+1 == n:
			trailing = true
			r++

		case p[r] == '.' && p[r+1] == '/':
			// . element
			r++

		case p[r] == '.' && p[r+1] == '.' && (r+2 == n || p[r+2] == '/'):
			// .. element: remove to last /
			r += 2

			if w > 1 {
				// can backtrack
				w--

				if buf == nil {
					for w > 1 && p[w] != '/' {
						w--
					}
				} else {
					for w > 1 && buf[w] != '/' {
						w--
					}
				}
			}

		default:
			// real path element.
			// add slash if needed
			if w > 1 {
				bufApp(&buf, p, w, '/')
				w++
			}

			// copy element
			for r < n && p[r] != '/' {
				bufApp(&buf, p, w, p[r])
				w++
				r++
			}
		}
	}

	// re-append trailing slash
	if trailing && w > 1 {
		bufApp(&buf, p, w, '/')
		w++
	}

	if buf == nil {
		return p[:w]
	}
	return string(buf[:w])
}

func bufApp(buf *[]byte, s string, w int, c byte) {
	if *buf == nil {
		if s[w] == c {
			return
		}

		*buf = make([]byte, len(s))
		copy(*buf, s[:w])
	}
	(*buf)[w] = c
}
