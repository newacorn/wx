// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package routing provides high performance and powerful HTTP routing capabilities.
package routing

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strconv"
	"sync"
	"time"

	"github.com/newacorn/fasthttp"
	"github.com/pkg/errors"
	"helpers/unsafefn"
)

type (
	// Handler is the function for handling HTTP requests.
	Handler func(*Ctx) error

	// Router manages routes and dispatches HTTP requests to the handlers of the matching routes.
	Router struct {
		server *fasthttp.Server
		// 路由组
		RouteGroup
		// *Ctx 缓存池
		pool sync.Pool
		// 路由路径或路由名称到路由的映射
		routes map[string]*Route
		// 请求方法 -> routeStore 的映射
		stores map[string]routeStore
		// 在从 Store 中操作路由时，会将 Ctx.pvalues 作为参数传递，变查找变填充。
		// 而此 pvalues 切片大小就是 maxParams。
		// 其值在将路由添加到 routeStore 中时进行更新，其值为已注册路径中那个参数数量最大的路由参数数量。
		maxParams int
		// 以路径+方法区分
		routeCount int
		// 通过 Router.NotFound 设置的值会覆盖此字段，将产生附加到 notFoundHandlers 中
		// 默认值是在 New 函数中初始化的 MethodNotAllowedHandler, NotFoundHandler
		notFound []Handler
		// Router.Use 注册的handlers + notFound
		// 档找不到路由时，会执行notFoundHandlers中的handler，从索引0开始执行。
		notFoundHandlers []Handler
		// /a是否互相匹配/a/
		strict bool
		// 路径是否区分大小写
		caseSensitive bool
	}

	// routeStore stores route paths and the corresponding handlers.
	routeStore interface {
		Add(key string, data interface{}) int
		Get(key string, pvalues []string) (data interface{}, pnames []string)
		String() string
	}
)

// Methods lists all supported HTTP methods by Router.
var Methods = []string{
	"CONNECT",
	"DELETE",
	"GET",
	"HEAD",
	"OPTIONS",
	"PATCH",
	"POST",
	"PUT",
	"TRACE",
}

// New creates a new Router object.
func New() *Router {
	r := &Router{
		server: &fasthttp.Server{
			// Logger:       &disableLogger{},
			LogAllErrors: false,
			// ErrorHandler: app.serverErrorHandler,
		},
		routes: make(map[string]*Route),
		stores: make(map[string]routeStore),
	}
	r.server.Handler = r.HandleRequest
	r.RouteGroup = *newRouteGroup("", r, make([]Handler, 0))
	r.NotFound(MethodNotAllowedHandler, NotFoundHandler)
	r.pool.New = func() interface{} {
		return &Ctx{
			pvalues: make([]string, r.maxParams),
			router:  r,
			bytes:   make([]byte, 0, 40),
		}
	}
	return r
}

// HandleRequest handles the HTTP request.
func (r *Router) HandleRequest(ctx *fasthttp.RequestCtx) {
	c := r.pool.Get().(*Ctx)
	c.init(ctx)
	c.handlers, c.pnames, c.route = r.find(string(ctx.Method()), unsafefn.BtoS(ctx.Path()), c.pvalues)
	if err := c.Next(); err != nil {
		r.handleError(c, err)
	}
	c.clear()
	r.pool.Put(c)
}

// Route returns the named route.
// Nil is returned if the named route cannot be found.
func (r *Router) Route(name string) *Route {
	return r.routes[name]
}

// Use appends the specified handlers to the router and shares them with all routes.
func (r *Router) Use(handlers ...Handler) {
	r.RouteGroup.Use(handlers...)
	r.notFoundHandlers = combineHandlers(r.handlers, r.notFound)
}

// NotFound specifies the handlers that should be invoked when the router cannot find any route matching a request.
// Note that the handlers registered via Use will be invoked first in this case.
func (r *Router) NotFound(handlers ...Handler) {
	r.notFound = handlers
	r.notFoundHandlers = combineHandlers(r.handlers, r.notFound)
}

// handleError is the error handler for handling any unhandled errors.
func (r *Router) handleError(c *Ctx, err error) {
	if httpError, ok := err.(HTTPError); ok {
		c.Error(httpError.Error(), httpError.StatusCode())
	} else {
		c.Error(err.Error(), http.StatusInternalServerError)
	}
}

type routeData struct {
	handlers []Handler
	route    *Route
}

func (r *Router) add(method, path string, handlers []Handler, route *Route) {
	store := r.stores[method]
	if store == nil {
		store = newStore()
		r.stores[method] = store
	}
	if n := store.Add(path, &routeData{route: route, handlers: handlers}); n > r.maxParams {
		r.maxParams = n
	}
}

func (r *Router) find(method, path string, pvalues []string) (handlers []Handler, pnames []string, route *Route) {
	var hh interface{}
	if store := r.stores[method]; store != nil {
		hh, pnames = store.Get(path, pvalues)
	}
	if hh != nil {
		r := hh.(*routeData)
		return r.handlers, pnames, r.route
	}
	return r.notFoundHandlers, pnames, nil
}

func (r *Router) findAllowedMethods(path string, ctx *Ctx) {
	// pvalues := make([]string, r.maxParams)
	for m, store := range r.stores {
		if handlers, _ := store.Get(path, ctx.pvalues); handlers != nil {
			ctx.bytes = append(ctx.bytes, m...)
			ctx.bytes = append(ctx.bytes, ',')
		}
	}
}

// NotFoundHandler returns a 404 HTTP error indicating a request has no matching route.
func NotFoundHandler(*Ctx) error {
	return NewHTTPError(http.StatusNotFound)
}

// MethodNotAllowedHandler handles the situation when a request has matching route without matching HTTP method.
// In this case, the handler will respond with an Allow HTTP header listing the allowed HTTP methods.
// Otherwise, the handler will do nothing and let the next handler (usually a NotFoundHandler) to handle the problem.
func MethodNotAllowedHandler(c *Ctx) error {
	c.Router().findAllowedMethods(unsafefn.BtoS(c.Path()), c)
	if len(c.bytes) == 0 {
		return nil
	}
	c.bytes = append(c.bytes, "OPTIONS"...)
	c.Response.Header.Set("Allow", unsafefn.BtoS(c.bytes))
	if string(c.Method()) != "OPTIONS" {
		c.Response.SetStatusCode(http.StatusMethodNotAllowed)
	}
	c.Abort()
	return nil
}
func (r *Router) Test(req *http.Request, msTimeout ...int) (*http.Response, error) {
	// Set timeout
	timeout := 1000
	if len(msTimeout) > 0 {
		timeout = msTimeout[0]
	}

	// Add Content-Length if not provided with body
	if req.Body != http.NoBody && req.Header.Get(HeaderContentLength) == "" {
		req.Header.Add(fasthttp.HeaderContentLength, strconv.FormatInt(req.ContentLength, 10))
	}

	// Dump raw http request
	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to dump request:")
	}

	// Create test connection
	conn := new(testConn)

	// Write raw http request
	if _, err := conn.r.Write(dump); err != nil {
		return nil, fmt.Errorf("failed to write: %w", err)
	}
	// prepare the server for the start

	// Serve conn to server
	channel := make(chan error)
	go func() {
		var returned bool
		defer func() {
			if !returned {
				channel <- fmt.Errorf("runtime.Goexit() called in handler or server panic")
			}
		}()

		channel <- r.server.ServeConn(conn)
		returned = true
	}()

	// Wait for callback
	if timeout >= 0 {
		// With timeout
		select {
		case err = <-channel:
		case <-time.After(time.Duration(timeout) * time.Millisecond):
			return nil, fmt.Errorf("test: timeout error %vms", timeout)
		}
	} else {
		// Without timeout
		err = <-channel
	}

	// Check for errors
	if err != nil && !errors.Is(err, fasthttp.ErrGetOnly) {
		return nil, err
	}

	// Read response
	buffer := bufio.NewReader(&conn.w)

	// Convert raw http response to *http.Response
	res, err := http.ReadResponse(buffer, req)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return res, nil
}
