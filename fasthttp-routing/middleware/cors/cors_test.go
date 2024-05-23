package cors

import (
	"regexp"
	"strings"
	"testing"

	routing "fasthttp-routing"
	"github.com/newacorn/fasthttp"
	"github.com/stretchr/testify/assert"
)

func Test_CORS_Defaults(t *testing.T) {
	t.Parallel()
	r := routing.New()
	r.Use(New())
	testDefaultOrEmptyConfig(t, r)
}
func testDefaultOrEmptyConfig(t *testing.T, r *routing.Router) {
	h := r.HandleRequest
	ctx := &fasthttp.RequestCtx{}
	a := assert.New(t)
	ctx.Request.Header.SetMethod(routing.MethodGet)
	h(ctx)
	a.Equal(routing.StatusNotFound, ctx.Response.StatusCode())
	a.Empty(ctx.Response.Header.Peek(routing.HeaderAccessControlAllowOrigin))
	a.Empty(ctx.Response.Header.Peek(routing.HeaderAccessControlAllowHeaders))
	a.Empty(ctx.Response.Header.Peek(routing.HeaderAccessControlAllowMethods))
	a.Empty(ctx.Response.Header.Peek(routing.HeaderAccessControlAllowCredentials))
	a.Empty(ctx.Response.Header.Peek(routing.HeaderAccessControlExposeHeaders))
	a.Empty(ctx.Response.Header.Peek(routing.HeaderAccessControlMaxAge))
	//
	ctx.Request.Reset()
	ctx.Response.Reset()
	ctx.Request.SetRequestURI("/api")
	ctx.Request.Header.SetMethod(routing.MethodOptions)
	methods := []string{routing.MethodGet}
	ctx.Request.Header.Set(routing.HeaderAccessControlAllowMethods, strings.Join(methods, ", "))
	h(ctx)
	a.Equal([]byte(nil), ctx.Response.Header.Peek(routing.HeaderAccessControlAllowOrigin))
	//
	ctx.Request.Reset()
	ctx.Response.Reset()
	ctx.Request.Header.Set(routing.HeaderOrigin, "http://localhost")
	ctx.Request.SetRequestURI("/api")
	ctx.Request.Header.SetMethod(routing.MethodOptions)
	h(ctx)
	a.Equal([]byte("http://localhost"), ctx.Response.Header.Peek(routing.HeaderAccessControlAllowOrigin))
}
func Test_CORS_WithExposeHeaders(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := &DefCfg
	cfg.ExposedHeaders = "X-My-Header, X-My-Header2"
	r.Use(New())
	testWithExposeHeaders(t, r, []byte("X-My-Header, X-My-Header2"))
}
func testWithExposeHeaders(t *testing.T, r *routing.Router, exposedHeaders []byte) {
	t.Helper()
	a := assert.New(t)
	h := r.HandleRequest
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api")
	ctx.Request.Header.SetMethod(routing.MethodOptions)
	ctx.Request.Header.Set(routing.HeaderOrigin, "http://localhsot")
	ctx.Request.Header.Set(routing.HeaderAccessControlRequestMethod, routing.MethodGet)
	h(ctx)
	a.Equal([]byte(nil), ctx.Response.Header.Peek(routing.HeaderAccessControlExposeHeaders))
	//
	ctx.Request.Reset()
	ctx.Response.Reset()
	ctx.Request.SetRequestURI("/api")
	ctx.Request.Header.SetMethod(routing.MethodGet)
	ctx.Request.Header.Set(routing.HeaderOrigin, "http://localhsot")
	h(ctx)
	a.Equal(exposedHeaders, ctx.Response.Header.Peek(routing.HeaderAccessControlExposeHeaders))
}
func Test_CORS_WithCredentialsIsTrue(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.SupportsCredentials = true
	r.Use(New(&cfg))
	testWithCredentials(t, r, []byte("true"))
}
func Test_CORS_WithCredentialsIsFalse(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	r.Use(New(&cfg))
	testWithCredentials(t, r, []byte(nil))
}
func testWithCredentials(t *testing.T, r *routing.Router, expect []byte) {
	t.Helper()
	a := assert.New(t)
	h := r.HandleRequest
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api")
	ctx.Request.Header.SetMethod(routing.MethodOptions)
	ctx.Request.Header.Set(routing.HeaderOrigin, "http://localhsot")
	ctx.Request.Header.Set(routing.HeaderAccessControlRequestMethod, routing.MethodGet)
	h(ctx)
	a.Equal(expect, ctx.Response.Header.Peek(routing.HeaderAccessControlAllowCredentials))
}
func Test_CORS_WithAllowedHeadersIsBytes(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.AllowedHeaders = []byte("X-My-Header, X-My-Header2")
	r.Use(New(&cfg))
	testWithAllowedHeaders(t, r, []byte("X-My-Header, X-My-Header2"), "")
}
func Test_CORS_WithAllowedHeadersIsTrue(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.AllowedHeaders = []byte{}
	r.Use(New(&cfg))
	testWithAllowedHeaders(t, r, []byte(nil), "")
	testWithAllowedHeaders(t, r, []byte("X-My-Header, X-My-Header2"), "X-My-Header, X-My-Header2")
}
func testWithAllowedHeaders(t *testing.T, r *routing.Router, expect []byte, requestHeaders string) {
	t.Helper()
	a := assert.New(t)
	h := r.HandleRequest
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api")
	ctx.Request.Header.SetMethod(routing.MethodOptions)
	if len(requestHeaders) != 0 {
		ctx.Request.Header.Set(routing.HeaderAccessControlRequestHeaders, requestHeaders)
	}
	ctx.Request.Header.Set(routing.HeaderOrigin, "http://localhsot")
	ctx.Request.Header.Set(routing.HeaderAccessControlRequestMethod, routing.MethodGet)
	h(ctx)
	a.Equal(expect, ctx.Response.Header.Peek(routing.HeaderAccessControlAllowHeaders))
}
func Test_CORS_WithAllowOriginsIsNotTrueAndNotNil(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.AllowedOrigins = []string{"http://localhost"}
	r.Use(New(&cfg))
	testWithAllowOrigins(t, r, "http://localhost", []byte("http://localhost"))
}
func Test_CORS_WithAllowOriginsIsTrue(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.AllowedOrigins = []string{}
	r.Use(New(&cfg))
	testWithAllowOrigins(t, r, "http://localhost", []byte("*"))
}
func Test_CORS_WithAllowOriginsIsTrueSupportCredentialsIsTrue(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.AllowedOrigins = []string{}
	cfg.SupportsCredentials = true
	r.Use(New(&cfg))
	testWithAllowOrigins(t, r, "http://localhost", []byte("http://localhost"))
}
func Test_CORS_WithAllowOriginsIsNil(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.AllowedOrigins = nil
	cfg.SupportsCredentials = true
	r.Use(New(&cfg))
	testWithAllowOrigins(t, r, "http://localhost", []byte(nil))
}
func testWithAllowOrigins(t *testing.T, r *routing.Router, origin string, expect []byte) {
	t.Helper()
	a := assert.New(t)
	h := r.HandleRequest
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api")
	ctx.Request.Header.SetMethod(routing.MethodOptions)
	ctx.Request.Header.Set(routing.HeaderOrigin, origin)
	ctx.Request.Header.Set(routing.HeaderAccessControlRequestMethod, routing.MethodGet)
	h(ctx)
	a.Equal(expect, ctx.Response.Header.Peek(routing.HeaderAccessControlAllowOrigin))
}
func Test_CORS_WithAllowedOriginsPatternsIsNotNil(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.AllowedOriginsPatterns = []*regexp.Regexp{regexp.MustCompile("http://localhost")}
	r.Use(New(&cfg))
	testWithAllowedOriginsPatterns(t, r, "http://localhost", []byte("http://localhost"))
}
func testWithAllowedOriginsPatterns(t *testing.T, r *routing.Router, origin string, expect []byte) {
	t.Helper()
	a := assert.New(t)
	h := r.HandleRequest
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api")
	ctx.Request.Header.SetMethod(routing.MethodOptions)
	ctx.Request.Header.Set(routing.HeaderOrigin, origin)
	ctx.Request.Header.Set(routing.HeaderAccessControlRequestMethod, routing.MethodGet)
	h(ctx)
	a.Equal(expect, ctx.Response.Header.Peek(routing.HeaderAccessControlAllowOrigin))
}
func Test_CORS_WithAllowedOriginsPatternsWithAllowOriginsIsAsterisk(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.AllowedOriginsPatterns = []*regexp.Regexp{regexp.MustCompile("123")}
	cfg.AllowedOrigins = []string{"*"}
	r.Use(New(&cfg))
	testWithAllowedOriginsPatterns(t, r, "http://localhost", []byte(nil))
}
func Test_CORS_WithAllowedOriginsPatternsDoesntMatchWithAllowOriginsIsTrueSupportCredentialsIsTrue(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.AllowedOriginsPatterns = []*regexp.Regexp{regexp.MustCompile("123")}
	cfg.SupportsCredentials = true
	cfg.AllowedOrigins = []string{"*"}
	r.Use(New(&cfg))
	testWithAllowedOriginsPatterns(t, r, "http://localhost", []byte(nil))
}
func Test_CORS_WithAllowedOriginsPatternsMatchWithAllowOriginsIsTrueSupportCredentialsIsTrue(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.AllowedOriginsPatterns = []*regexp.Regexp{regexp.MustCompile("^.+")}
	cfg.SupportsCredentials = true
	cfg.AllowedOrigins = []string{}
	r.Use(New(&cfg))
	testWithAllowedOriginsPatterns(t, r, "http://localhost", []byte("http://localhost"))
}
func Test_CORS_WithAllowedOriginsPatternsMatchWithAllowOriginsIsTrueSupportCredentialsIsFalse(t *testing.T) {
	t.Parallel()
	r := routing.New()
	cfg := DefCfg
	cfg.AllowedOriginsPatterns = []*regexp.Regexp{regexp.MustCompile("http://localhost")}
	cfg.SupportsCredentials = false
	cfg.AllowedOrigins = []string{}
	r.Use(New(&cfg))
	testWithAllowedOriginsPatterns(t, r, "http://localhost", []byte("*"))
}
