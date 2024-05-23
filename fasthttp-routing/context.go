// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package routing

import (
	"fmt"
	"net"
	"strconv"

	"github.com/newacorn/fasthttp"
	"github.com/rs/zerolog"
	"helpers/unsafefn"
	"helpers/utilnet"
)

// SerializeFunc serializes the given data of arbitrary type into a byte array.
type SerializeFunc func(data interface{}) ([]byte, error)
type XFFInfo interface {
	Host(ctx *Ctx)
	Port(ctx *Ctx)
	IP(ctx *Ctx)
	Secure(ctx *Ctx)
}
type App struct {
	Log *zerolog.Logger
}

const (
	PROTOUNKNOW   = 0
	PROTOUNSECURE = 1
	PROTOSECURE   = 2
)

// Ctx represents the contextual data and environment while processing an incoming HTTP request.
type Ctx struct {
	App *App
	*fasthttp.RequestCtx
	Serialize SerializeFunc // the function serializing the given data of arbitrary type into a byte array.

	XFFInfo XFFInfo
	//
	RSecure int
	//
	RIP   []byte
	RPort int
	RHost []byte

	router   *Router
	pnames   []string               // list of route parameter names
	pvalues  []string               // list of parameter values corresponding to pnames
	data     map[string]interface{} // data items managed by Get and Set
	index    int                    // the index of the currently executing handler in handlers
	handlers []Handler              // the handlers associated with the current route
	route    *Route
	bytes    []byte
}

// Router returns the Router that is handling the incoming HTTP request.
func (c *Ctx) Router() *Router {
	return c.router
}
func (c *Ctx) Route() *Route {
	return c.route
}

// Param returns the named parameter value that is found in the URL path matching the current route.
// If the named parameter cannot be found, an empty string will be returned.
func (c *Ctx) Param(name string) string {
	for i, n := range c.pnames {
		if n == name {
			return c.pvalues[i]
		}
	}
	return ""
}
func (c *Ctx) Host() (host []byte) {
	if c.XFFInfo != nil {
		if len(c.RHost) != 0 {
			return c.RHost
		}
		c.XFFInfo.Host(c)
		if len(c.RHost) != 0 {
			return c.RHost
		}
	}

	h, _, _ := utilnet.SplitIpAndPort(unsafefn.BtoS(c.Request.Header.Host()))
	return unsafefn.StoB(h)
}
func (c *Ctx) IP() (ip []byte) {
	if c.XFFInfo != nil {
		if len(c.RIP) != 0 {
			return c.RIP
		}
		c.XFFInfo.IP(c)
		if len(c.RIP) != 0 {
			return c.RIP
		}
	}
	r := c.RemoteAddr()
	r1, ok := r.(*net.TCPAddr)
	if ok {
		return unsafefn.StoB(r1.IP.String())
	}
	r2, ok := r.(*net.UDPAddr)
	if ok {
		return unsafefn.StoB(r2.IP.String())
	}
	p, _, _ := utilnet.SplitIpAndPort(c.RemoteAddr().String())
	return unsafefn.StoB(p)
}
func (c *Ctx) Port() (port int) {
	if c.XFFInfo != nil {
		if c.RPort > 0 {
			return c.RPort
		}
		c.XFFInfo.Port(c)
		if c.RPort < 0 {
			c.RPort = -c.RPort
		}
		if c.RPort != 0 {
			return c.RPort
		}
	}
	_, p, err := utilnet.SplitIpAndPort(unsafefn.BtoS(c.Request.Header.Host()))
	if err == nil && len(p) != 0 {
		port, err = strconv.Atoi(p)
		if err == nil {
			return port
		}
	}
	if c.Secure() {
		return 443
	}
	return 80
}
func (c *Ctx) Secure() (secure bool) {
	if c.XFFInfo != nil {
		if c.RSecure != PROTOUNKNOW {
			if c.RSecure == PROTOSECURE {
				return true
			}
			return false
		}
		c.XFFInfo.Secure(c)
		if c.RSecure != PROTOUNKNOW {
			if c.RSecure == PROTOSECURE {
				return true
			}
			return false
		}
	}
	return c.IsTLS()
}
func (c *Ctx) Proto() (proto string) {
	if c.Secure() {
		return HTTPS
	}
	return HTTP
}

// Get returns the named data item previously registered with the context by calling Set.
// If the named data item cannot be found, nil will be returned.
func (c *Ctx) Get(name string) interface{} {
	return c.data[name]
}

// Set stores the named data item in the context so that it can be retrieved later.
func (c *Ctx) Set(name string, value interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[name] = value
}

// Next calls the rest of the handlers associated with the current route.
// If any of these handlers returns an error, Next will return the error and skip the following handlers.
// Next is normally used when a handler needs to do some postprocessing after the rest of the handlers
// are executed.
// 即使 Ctx.Next 不被调用，剩余的handler也可以被访问，除非直接返回不为nil的err或者调用Ctx.Abort方法。
func (c *Ctx) Next() error {
	c.index++
	for n := len(c.handlers); c.index < n; c.index++ {
		if err := c.handlers[c.index](c); err != nil {
			return err
		}
	}
	return nil
}
func (c *Ctx) Error(msg string, statusCode int) {
	c.Response.ResetBody()
	c.SetStatusCode(statusCode)
	if len(c.RequestCtx.Response.Header.ContentType()) == 0 {
		c.SetContentTypeBytes(defaultContentType)
	}
	c.SetBodyString(msg)
}

// Abort skips the rest of the handlers associated with the current route.
// Abort is normally used when a handler handles the request normally and wants to skip the rest of the handlers.
// If a handler wants to indicate an error condition, it should simply return the error without calling Abort.
func (c *Ctx) Abort() {
	c.index = len(c.handlers)
}

// URL creates a URL using the named route and the parameter values.
// The parameters should be given in the sequence of name1, value1, name2, value2, and so on.
// If a parameter in the route is not provided a value, the parameter token will remain in the resulting URL.
// Parameter values will be properly URL encoded.
// The method returns an empty string if the URL creation fails.
func (c *Ctx) URL(route string, pairs ...interface{}) string {
	if r := c.router.routes[route]; r != nil {
		return r.URL(pairs...)
	}
	return ""
}

// WriteData writes the given data of arbitrary type to the response.
// The method calls the Serialize() method to convert the data into a byte array and then writes
// the byte array to the response.
func (c *Ctx) WriteData(data interface{}) (err error) {
	var bytes []byte
	if bytes, err = c.Serialize(data); err == nil {
		_, err = c.Write(bytes)
	}
	return
}

// init sets the request and response of the context and resets all other properties.
func (c *Ctx) init(ctx *fasthttp.RequestCtx) {
	c.RequestCtx = ctx
	c.index = -1
	c.Serialize = Serialize
}
func (c *Ctx) clear() {
	c.data = nil
	c.route = nil
}

// Serialize converts the given data into a byte array.
// If the data is neither a byte array nor a string, it will call fmt.Sprint to convert it into a string.
func Serialize(data interface{}) (bytes []byte, err error) {
	switch data.(type) {
	case []byte:
		return data.([]byte), nil
	case string:
		return []byte(data.(string)), nil
	default:
		if data != nil {
			return []byte(fmt.Sprint(data)), nil
		}
	}
	return nil, nil
}
