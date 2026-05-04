package server

import (
	"fmt"
	"strings"
)

// HandlerFunc is the function signature that all route handlers and middleware
// must match. req carries the parsed HTTP request. res is written to by the
// handler and flushed to the client after the handler returns.
type HandlerFunc func(req *Request, res *Response)

// node is one segment in the Trie
// e.g. the path /users/:id has three nodes:
//
//	root → "users" → ":id"
type node struct {
	segment   string                 // the path segment this node represents e.g. "users" or ":id"
	children  []*node                // child nodes — one per unique next segment
	handlers  map[string]HandlerFunc // method → handler e.g. "GET" → getUserHandler
	isParam   bool                   // true if this segment is a wildcard e.g. :id
	paramName string                 // the name of the param e.g. "id" from ":id"
}

// newNode creates an empty Trie node for a given segment
func newNode(segment string) *node {
	n := &node{
		segment:  segment,
		handlers: make(map[string]HandlerFunc),
	}
	// if the segment starts with : it is a path parameter
	if strings.HasPrefix(segment, ":") {
		n.isParam = true
		n.paramName = segment[1:] // strip the colon — ":id" becomes "id"
	}
	return n
}

// Router holds the root of the Trie and dispatches incoming requests to the
// correct handler based on method and path. Create one with NewRouter().
type Router struct {
	root *node
}

// NewRouter creates a Router with an empty root Trie node.
func NewRouter() *Router {
	return &Router{
		root: newNode("/"),
	}
}

// add registers a handler for the given HTTP method and path in the Trie.
// Path segments prefixed with : are wildcard params e.g. "/users/:id".
// Registering the same method+path twice overwrites the first handler.
func (r *Router) add(method, path string, handler HandlerFunc) {
	// split the path into segments
	// "/users/:id" → ["users", ":id"]
	segments := splitPath(path)

	current := r.root

	for _, segment := range segments {
		child := findChild(current, segment)
		if child == nil {
			// this segment does not exist yet — create a new node
			child = newNode(segment)
			current.children = append(current.children, child)
		}
		current = child
	}

	// we have reached the node for the final segment
	// store the handler under this HTTP method
	current.handlers[method] = handler
}

// dispatch finds the right handler for a request and calls it
// it also extracts path parameters e.g. /users/42 → params["id"] = "42"
func (r *Router) dispatch(req *Request) *Response {
	segments := splitPath(req.Path)
	params := make(map[string]string)

	node := matchNode(r.root, segments, params)

	res := newResponse()

	if node == nil {
		// no node matched this path at all
		return res.Status(404).Body(fmt.Sprintf("404 Not Found: %s", req.Path))
	}

	handler, ok := node.handlers[req.Method]
	if !ok {
		// path matched but no handler for this method
		return res.Status(405).Body(fmt.Sprintf("405 Method Not Allowed: %s %s", req.Method, req.Path))
	}

	// attach extracted params to the request so handlers can use them
	req.Params = params

	// call the matched handler
	handler(req, res)

	return res
}

// matchNode recursively walks the Trie to find a matching node
// it populates params as it encounters wildcard segments
func matchNode(current *node, segments []string, params map[string]string) *node {
	// no more segments — current node is the match
	if len(segments) == 0 {
		return current
	}

	segment := segments[0]
	rest := segments[1:]

	for _, child := range current.children {
		if child.isParam {
			// wildcard match — any segment matches, extract the value
			params[child.paramName] = segment
			result := matchNode(child, rest, params)
			if result != nil {
				return result
			}
			// if deeper match failed, undo the param extraction
			delete(params, child.paramName)
		} else if child.segment == segment {
			// exact match
			result := matchNode(child, rest, params)
			if result != nil {
				return result
			}
		}
	}

	// no child matched
	return nil
}

// findChild looks for an existing child node with the given segment
func findChild(n *node, segment string) *node {
	for _, child := range n.children {
		if child.segment == segment {
			return child
		}
	}
	return nil
}

// splitPath splits a URL path into segments, ignoring empty parts
// "/users/:id" → ["users", ":id"]
// "/"          → []
func splitPath(path string) []string {
	parts := strings.Split(path, "/")
	segments := []string{}
	for _, part := range parts {
		if part != "" {
			segments = append(segments, part)
		}
	}
	return segments
}
