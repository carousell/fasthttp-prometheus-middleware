// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

// Package router is a trie based high performance HTTP request router.
//
// A trivial example is:
//
// package main

// import (
//     "fmt"
//     "log"
//
//     "github.com/fasthttp/router"
//     "github.com/valyala/fasthttp"
// )

// func Index(ctx *fasthttp.RequestCtx) {
//     fmt.Fprint(ctx, "Welcome!\n")
// }

// func Hello(ctx *fasthttp.RequestCtx) {
//     fmt.Fprintf(ctx, "hello, %s!\n", ctx.UserValue("name"))
// }

// func main() {
//     r := router.New()
//     r.GET("/", Index)
//     g := r.Group("/foo", Index)
//     g.GET("/bar", Index)
//     r.GET("/hello/:name", Hello)

//     log.Fatal(fasthttp.ListenAndServe(":8080", r.Handler))
// }
//
// The router matches incoming requests by the request method and the path.
// If a handle is registered for this path and method, the router delegates the
// request to that function.
// For the methods GET, POST, PUT, PATCH and DELETE shortcut functions exist to
// register handles, for all other methods router.Handle can be used.
//
// The registered path, against which the router matches incoming requests, can
// contain two types of parameters:
//  Syntax    Type
//  :name     named parameter
//  *name     catch-all parameter
//
// Named parameters are dynamic path segments. They match anything until the
// next '/' or the path end:
//  Path: /blog/:category/:post
//
//  Requests:
//   /blog/go/request-routers            match: category="go", post="request-routers"
//   /blog/go/request-routers/           no match, but the router would redirect
//   /blog/go/                           no match
//   /blog/go/request-routers/comments   no match
//
// Catch-all parameters match anything until the path end, including the
// directory index (the '/' before the catch-all). Since they match anything
// until the end, catch-all parameters must always be the final path element.
//  Path: /files/*filepath
//
//  Requests:
//   /files/                             match: filepath="/"
//   /files/LICENSE                      match: filepath="/LICENSE"
//   /files/templates/article.html       match: filepath="/templates/article.html"
//   /files                              no match, but the router would redirect
//
// The value of parameters is inside ctx.UserValue
// To retrieve the value of a parameter:
//  // use the name of the parameter
//  user := ps.UserValue("user")
//

package router

import (
	"strings"

	"github.com/savsgio/gotils"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
)

var (
	defaultContentType = []byte("text/plain; charset=utf-8")
	questionMark       = []byte("?")
)

// Router is a http.Handler which can be used to dispatch requests to different
// handler functions via configurable routes
type Router struct {
	parent          *Router
	beginPath       string
	trees           map[string]*node
	registeredPaths map[string][]string

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 307 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the router tries to fix the current request path, if no
	// handle is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a handle can be found for this route, the router makes a redirection
	// to the corrected path with status code 301 for GET requests and 307 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.
	RedirectFixedPath bool

	// If enabled, the router checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool

	// If enabled, the router automatically replies to OPTIONS requests.
	// Custom OPTIONS handlers take priority over automatic replies.
	HandleOPTIONS bool

	// Configurable http.Handler which is called when no matching route is
	// found. If it is not set, http.NotFound is used.
	NotFound fasthttp.RequestHandler

	// Configurable http.Handler which is called when a request
	// cannot be routed and HandleMethodNotAllowed is true.
	// If it is not set, http.Error with http.StatusMethodNotAllowed is used.
	// The "Allow" header with allowed request methods is set before the handler
	// is called.
	MethodNotAllowed fasthttp.RequestHandler

	// Function to handle panics recovered from http handlers.
	// It should be used to generate a error page and return the http error code
	// 500 (Internal Server Error).
	// The handler can be used to keep your server from crashing because of
	// unrecovered panics.
	PanicHandler func(*fasthttp.RequestCtx, interface{})
}

// New returns a new initialized Router.
// Path auto-correction, including trailing slashes, is enabled by default.
func New() *Router {
	return &Router{
		beginPath:              "/",
		trees:                  make(map[string]*node),
		registeredPaths:        make(map[string][]string),
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleMethodNotAllowed: true,
		HandleOPTIONS:          true,
	}
}

// Group returns a new grouped Router.
// Path auto-correction, including trailing slashes, is enabled by default.
func (r *Router) Group(path string) *Router {
	g := New()
	g.parent = r
	g.beginPath = path

	return g
}

// GET is a shortcut for router.Handle("GET", path, handle)
func (r *Router) GET(path string, handle fasthttp.RequestHandler) {
	r.Handle("GET", path, handle)
}

// HEAD is a shortcut for router.Handle("HEAD", path, handle)
func (r *Router) HEAD(path string, handle fasthttp.RequestHandler) {
	r.Handle("HEAD", path, handle)
}

// OPTIONS is a shortcut for router.Handle("OPTIONS", path, handle)
func (r *Router) OPTIONS(path string, handle fasthttp.RequestHandler) {
	r.Handle("OPTIONS", path, handle)
}

// POST is a shortcut for router.Handle("POST", path, handle)
func (r *Router) POST(path string, handle fasthttp.RequestHandler) {
	r.Handle("POST", path, handle)
}

// PUT is a shortcut for router.Handle("PUT", path, handle)
func (r *Router) PUT(path string, handle fasthttp.RequestHandler) {
	r.Handle("PUT", path, handle)
}

// PATCH is a shortcut for router.Handle("PATCH", path, handle)
func (r *Router) PATCH(path string, handle fasthttp.RequestHandler) {
	r.Handle("PATCH", path, handle)
}

// DELETE is a shortcut for router.Handle("DELETE", path, handle)
func (r *Router) DELETE(path string, handle fasthttp.RequestHandler) {
	r.Handle("DELETE", path, handle)
}

// Handle registers a new request handle with the given path and method.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) Handle(method, path string, handle fasthttp.RequestHandler) {
	if path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}

	if r.beginPath != "/" {
		path = r.beginPath + path
	}

	r.registeredPaths[method] = append(r.registeredPaths[method], path)

	// Call to the parent recursively until main router to register paths in it
	if r.parent != nil {
		r.parent.Handle(method, path, handle)
		return
	}

	root := r.trees[method]
	if root == nil {
		root = new(node)
		r.trees[method] = root
	}

	optionalPaths := getOptionalPaths(path)

	// if not has optional paths, adds the original
	if len(optionalPaths) == 0 {
		root.addRoute(path, handle)
	} else {
		for _, p := range optionalPaths {
			root.addRoute(p, handle)
		}
	}
}

// ServeFiles serves files from the given file system root.
// The path must end with "/*filepath", files are then served from the local
// path /defined/root/dir/*filepath.
// For example if root is "/etc" and *filepath is "passwd", the local file
// "/etc/passwd" would be served.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
//     router.ServeFiles("/src/*filepath", "/var/www")
func (r *Router) ServeFiles(path string, rootPath string) {
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("path must end with /*filepath in path '" + path + "'")
	}

	if r.beginPath != "/" {
		path = r.beginPath + path
	}

	if r.parent != nil {
		r.parent.ServeFiles(path, rootPath)
		return
	}

	prefix := path[:len(path)-10]
	fileHandler := fasthttp.FSHandler(rootPath, strings.Count(prefix, "/"))

	r.GET(path, func(ctx *fasthttp.RequestCtx) {
		fileHandler(ctx)
	})
}

// ServeFilesCustom serves files from the given file system settings.
// The path must end with "/*filepath", files are then served from the local
// path /defined/root/dir/*filepath.
// For example if root is "/etc" and *filepath is "passwd", the local file
// "/etc/passwd" would be served.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
//     router.ServeFilesCustom("/src/*filepath", *customFS)
func (r *Router) ServeFilesCustom(path string, fs *fasthttp.FS) {
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("path must end with /*filepath in path '" + path + "'")
	}

	if r.beginPath != "/" {
		path = r.beginPath + path
	}

	if r.parent != nil {
		r.parent.ServeFilesCustom(path, fs)
		return
	}

	prefix := path[:len(path)-10]
	stripSlashes := strings.Count(prefix, "/")

	if fs.PathRewrite == nil && stripSlashes > 0 {
		fs.PathRewrite = fasthttp.NewPathSlashesStripper(stripSlashes)
	}
	fileHandler := fs.NewRequestHandler()

	r.GET(path, func(ctx *fasthttp.RequestCtx) {
		fileHandler(ctx)
	})
}

// Handler makes the router implement the fasthttp.ListenAndServe interface.
func (r *Router) Handler(ctx *fasthttp.RequestCtx) {
	if r.PanicHandler != nil {
		defer r.recv(ctx)
	}

	path := gotils.B2S(ctx.Path())
	method := gotils.B2S(ctx.Method())

	if root := r.trees[method]; root != nil {
		if f, tsr := root.getValue(path, ctx); f != nil {
			f(ctx)
			return
		} else if method != "CONNECT" && path != "/" {
			code := 301 // Permanent redirect, request with GET method
			if method != "GET" {
				// Temporary redirect, request with same method
				// As of Go 1.3, Go does not support status code 308.
				code = 307
			}

			if tsr && r.RedirectTrailingSlash {
				uri := bytebufferpool.Get()

				if len(path) > 1 && path[len(path)-1] == '/' {
					uri.SetString(path[:len(path)-1])
				} else {
					uri.SetString(path)
					uri.WriteString("/")
				}

				if len(ctx.URI().QueryString()) > 0 {
					uri.WriteString("?")
					uri.Write(ctx.QueryArgs().QueryString())
				}

				ctx.Redirect(uri.String(), code)

				bytebufferpool.Put(uri)

				return
			}

			// Try to fix the request path
			if r.RedirectFixedPath {
				cpb := acquireCleanPathBuffer()
				cleanPathWithBuffer(cpb, path)
				fixedPath, found := root.findCaseInsensitivePath(gotils.B2S(cpb.buf), r.RedirectTrailingSlash)
				releaseCleanPathBuffer(cpb)

				if found {
					queryBuf := ctx.URI().QueryString()
					if len(queryBuf) > 0 {
						fixedPath = append(fixedPath, questionMark...)
						fixedPath = append(fixedPath, queryBuf...)
					}
					ctx.RedirectBytes(fixedPath, code)
					return
				}
			}
		}
	}

	if method == "OPTIONS" {
		// Handle OPTIONS requests
		if r.HandleOPTIONS {
			if allow := r.allowed(path, method); len(allow) > 0 {
				ctx.Response.Header.Set("Allow", allow)
				return
			}
		}
	} else {
		// Handle 405
		if r.HandleMethodNotAllowed {
			if allow := r.allowed(path, method); len(allow) > 0 {
				ctx.Response.Header.Set("Allow", allow)
				if r.MethodNotAllowed != nil {
					r.MethodNotAllowed(ctx)
				} else {
					ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
					ctx.SetContentTypeBytes(defaultContentType)
					ctx.SetBodyString(fasthttp.StatusMessage(fasthttp.StatusMethodNotAllowed))
				}
				return
			}
		}
	}

	// Handle 404
	if r.NotFound != nil {
		r.NotFound(ctx)
	} else {
		ctx.Error(fasthttp.StatusMessage(fasthttp.StatusNotFound), fasthttp.StatusNotFound)
	}
}

// Lookup allows the manual lookup of a method + path combo.
// This is e.g. useful to build a framework around this router.
// If the path was found, it returns the handle function and the path parameter
// values. Otherwise the third return value indicates whether a redirection to
// the same path with an extra / without the trailing slash should be performed.
func (r *Router) Lookup(method, path string, ctx *fasthttp.RequestCtx) (fasthttp.RequestHandler, bool) {
	if root := r.trees[method]; root != nil {
		return root.getValue(path, ctx)
	}
	return nil, false
}

// List returns all registered routes grouped by method
func (r *Router) List() map[string][]string {
	return r.registeredPaths
}

func (r *Router) allowed(path, reqMethod string) (allow string) {
	if path == "*" || path == "/*" { // server-wide
		for method := range r.trees {
			if method == "OPTIONS" {
				continue
			}

			// add request method to list of allowed methods
			if len(allow) == 0 {
				allow = method
			} else {
				allow += ", " + method
			}
		}
	} else { // specific path
		for method := range r.trees {
			// Skip the requested method - we already tried this one
			if method == reqMethod || method == "OPTIONS" {
				continue
			}

			handle, _ := r.trees[method].getValue(path, nil)
			if handle != nil {
				// add request method to list of allowed methods
				if len(allow) == 0 {
					allow = method
				} else {
					allow += ", " + method
				}
			}
		}
	}
	if len(allow) > 0 {
		allow += ", OPTIONS"
	}
	return
}

func (r *Router) recv(ctx *fasthttp.RequestCtx) {
	if rcv := recover(); rcv != nil {
		r.PanicHandler(ctx, rcv)
	}
}
