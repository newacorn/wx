// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package routing

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Route represents a URL path pattern that can be used to match requested URLs.
// 一个路径对应一个路由。
// 一个路由可以包含多个方法。
type Route struct {
	group      *RouteGroup
	name, path string
	template   string
	// methods 与 handlers 按索引对应
	methods  []string
	handlers [][]Handler
	// 路径中正则模式的参数
	regexps []*regexp.Regexp
	// 路径片段按路由参数分隔，路由参数统一格式为 <paraName:>/<paraName:\d+> -> <paraName>
	// /* -> /<*:.*> -> <*>
	segments []string
	// 路由参数的名称，与regexps按索引对应，如果参数不是正则模式则对应的regexp.Regexp是nil。
	paramNames []string
}

func (r *Route) Path() string {
	return r.path
}

// newRoute creates a new Route with the given route path and route group.
func newRoute(path string, group *RouteGroup) *Route {
	path = group.prefix + path
	name := path

	// an asterisk at the end matches any number of characters
	if strings.HasSuffix(path, "*") {
		path = path[:len(path)-1] + "<*:.*>"
	}

	route := &Route{
		group:    group,
		name:     name,
		path:     path,
		template: buildURLTemplate(path),
	}
	route.segments, route.paramNames, route.regexps = buildURLTemplate2(path)
	group.router.routes[name] = route

	return route
}

// Name sets the name of the route.
// This method will update the registration of the route in the router as well.
func (r *Route) Name(name string) *Route {
	r.name = combineNames(r.group.name, name)
	r.group.router.routes[r.name] = r
	return r
}

// Get adds the route to the router using the GET HTTP method.
func (r *Route) Get(handlers ...Handler) *Route {
	return r.add("GET", handlers)
}

// Post adds the route to the router using the POST HTTP method.
func (r *Route) Post(handlers ...Handler) *Route {
	return r.add("POST", handlers)
}

// Put adds the route to the router using the PUT HTTP method.
func (r *Route) Put(handlers ...Handler) *Route {
	return r.add("PUT", handlers)
}

// Patch adds the route to the router using the PATCH HTTP method.
func (r *Route) Patch(handlers ...Handler) *Route {
	return r.add("PATCH", handlers)
}

// Delete adds the route to the router using the DELETE HTTP method.
func (r *Route) Delete(handlers ...Handler) *Route {
	return r.add("DELETE", handlers)
}

// Connect adds the route to the router using the CONNECT HTTP method.
func (r *Route) Connect(handlers ...Handler) *Route {
	return r.add("CONNECT", handlers)
}

// Head adds the route to the router using the HEAD HTTP method.
func (r *Route) Head(handlers ...Handler) *Route {
	return r.add("HEAD", handlers)
}

// Options adds the route to the router using the OPTIONS HTTP method.
func (r *Route) Options(handlers ...Handler) *Route {
	return r.add("OPTIONS", handlers)
}

// Trace adds the route to the router using the TRACE HTTP method.
func (r *Route) Trace(handlers ...Handler) *Route {
	return r.add("TRACE", handlers)
}

// To adds the route to the router with the given HTTP methods and handlers.
// Multiple HTTP methods should be separated by commas (without any surrounding spaces).
func (r *Route) To(methods string, handlers ...Handler) *Route {
	for _, method := range strings.Split(methods, ",") {
		r.add(method, handlers)
	}
	return r
}

// URL creates a URL using the current route and the given parameters.
// The parameters should be given in the sequence of name1, value1, name2, value2, and so on.
// If a parameter in the route is not provided a value, the parameter token will remain in the resulting URL.
// The method will perform URL encoding for all given parameter values.
func (r *Route) URL(pairs ...interface{}) (s string) {
	s = r.template
	for i := 0; i < len(pairs); i++ {
		name := fmt.Sprintf("<%v>", pairs[i])
		value := ""
		if i < len(pairs)-1 {
			value = url.QueryEscape(fmt.Sprint(pairs[i+1]))
		}
		s = strings.Replace(s, name, value, -1)
	}
	return
}

// add registers the route, the specified HTTP method and the handlers to the router.
// The handlers will be combined with the handlers of the route group.
func (r *Route) add(method string, handlers []Handler) *Route {
	r.group.router.routeCount++
	hh := combineHandlers(r.group.handlers, handlers)
	r.methods = append(r.methods, method)
	r.handlers = append(r.handlers, hh)
	r.group.router.add(method, r.path, hh, r)
	return r
}
func (r *Route) URLByNames(names []string) (url string, err error) {
	if len(names)/2 != len(r.paramNames) {
		err = errors.New("paramNames count error: need " + strconv.Itoa(len(r.paramNames)) + ",got " + strconv.Itoa(len(names)/2))
		return
	}
	var matched bool
	for i, n := range names {
		matched = false
		if i%2 == 0 {
			for i1, n1 := range r.paramNames {
				if n == n1 {
					if r.regexps[i1] == nil {
						if strings.Contains(names[i+1], "/") {
							err = errors.New(names[i+1] + " contains " + "/")
							return
						}
					} else {
						if !r.regexps[i1].MatchString(names[i+1]) {
							err = errors.New("param: " + names[i+1] + " doesn't match " + r.regexps[i].String())
							return
						}
					}
					matched = true
					break
				}
			}
			if !matched {
				err = errors.New("doesn't contain paramName:" + n)
				return
			}
		}
		continue
	}
	for _, s := range r.segments {
		if s[0] == '<' {
			for i := range names {
				if s[1:len(s)-1] == names[i] {
					url += names[i+1]
					break
				}
			}
			continue
		}
		url += s
	}
	return
}
func (r *Route) URLByIndex(params []string) (url string, err error) {
	if len(params) != len(r.paramNames) {
		err = errors.New("paramNames count error: need " + strconv.Itoa(len(r.paramNames)) + ",got " + strconv.Itoa(len(params)))
		return
	}
	for i := range params {
		if r.regexps[i] == nil {
			if strings.Contains(params[i], "/") {
				err = errors.New(params[i] + " contains " + "/")
				return
			}
			continue
		}
		if !r.regexps[i].MatchString(params[i]) {
			err = errors.New("param: " + params[i] + " doesn't match " + r.regexps[i].String())
			return
		}
	}
	j := 0
	for _, s := range r.segments {
		if s[0] == '<' {
			url += params[j]
			j++
			continue
		}
		url += s
	}
	return
}
func buildURLTemplate2(path string) (segments []string, paramNames []string, regexps []*regexp.Regexp) {
	start, end := -1, -1
	for i := 0; i < len(path); i++ {
		if path[i] == '<' && start < 0 {
			start = i
		} else if path[i] == '>' && start >= 0 {
			name := path[start+1 : i]
			var reg *regexp.Regexp
			for j := start + 1; j < i; j++ {
				if path[j] == ':' {
					name = path[start+1 : j]
					if path[j+1:i] != "" {
						reg = regexp.MustCompile(`^` + path[j+1:i])
					}
					break
				}
			}
			paramNames = append(paramNames, name)
			regexps = append(regexps, reg)
			segments = append(segments, path[end+1:start], "<"+name+">")
			end = i
			start = -1
		}
	}
	if end < 0 {
		segments = append(segments, path)
	} else if end < len(path)-1 {
		segments = append(segments, path[end+1:])
	}
	/*
		if path[len(path)-1] == '*' {
			l := len(segments)
			regexps = append(regexps, regexp.MustCompile(`^.*`))
			paramNames = append(paramNames, `*`)
			last := segments[l-1]
			last = last[:len(last)-1]
			segments[l-1] = last
			segments = append(segments, `<*>`)
		}
	*/
	return
}

// buildURLTemplate converts a route pattern into a URL template by removing regular expressions in parameter tokens.
func buildURLTemplate(path string) string {
	template, start, end := "", -1, -1
	for i := 0; i < len(path); i++ {
		if path[i] == '<' && start < 0 {
			start = i
		} else if path[i] == '>' && start >= 0 {
			name := path[start+1 : i]
			for j := start + 1; j < i; j++ {
				if path[j] == ':' {
					name = path[start+1 : j]
					break
				}
			}
			template += path[end+1:start] + "<" + name + ">"
			end = i
			start = -1
		}
	}
	if end < 0 {
		template = path
	} else if end < len(path)-1 {
		template += path[end+1:]
	}
	return template
}

// combineHandlers merges two lists of handlers into a new list.
func combineHandlers(h1 []Handler, h2 []Handler) []Handler {
	hh := make([]Handler, len(h1)+len(h2))
	copy(hh, h1)
	copy(hh[len(h1):], h2)
	return hh
}
