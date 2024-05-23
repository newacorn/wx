package cors

import (
	"regexp"
	"slices"
	"strings"

	routing "fasthttp-routing"
	"helpers/unsafefn"
)

/**
// translate from laravel's cors middleware
  'paths' => ['api/*', 'sanctum/csrf-cookie'],

  'allowed_methods' => ['*'],

  'allowed_origins' => ['*'],

  'allowed_origins_patterns' => [],

  'allowed_headers' => ['*'],

  'exposed_headers' => [],

  'max_age' => 0,

  'supports_credentials' => false,
*/

// Config Access-Control-Allow-Origin 来源值优先级：
//  1. 当 AllowOrigins==true && SupportCredentials==false时，肯定是 *
//  2. 当 AllowOrigins=[]string{"*"}且len(AllowOriginsPatterns)==0时，肯定是请求的头Origin (如果请求头为空不设置)
//  3. 当 len(AllowOriginsPatters)!=0 时，且1不成立时，会遍历 AllowOriginsPatterns(不为true)和AllowOriginsPatters去匹配
//     请求的Origin，如果匹配成功者设置我请求头的Origin。
//
// Config defines the config for middleware.
type Config struct {
	Skip func(c *routing.Ctx) bool
	// []string{"*"} match all paths
	// 其它值做前缀匹配
	//
	// 前缀匹配，不匹配即跳过处理
	Path []string
	// []byte{} represent true 会将Access-Control-Allow-Method 设置为请求的对应header
	// 其它将 Access-Control-Allow-Method 设置为此字段值
	//
	// 在预请求中设置
	AllowedMethods []byte
	// []string{} represent true
	//
	// 0. AllowedOrigins=[]string{*} 接收所有origin且Access-Control-Allow-Origin设置为请求origin
	// 1. AllowedOrigins==true && SupportsCredentials==false: Access-Control-Allow-Origin设置为*
	// 2. len(AllowedOrigins)==1&&len(AllowedOriginsPatterns) ==0: Access-Control-Allow-Origin设置为请求值
	// 3. 请求头中包含Origin && 请求Origin在AllowedOrigins中或者请求Origin匹配AllowedOriginsPatterns: Access-Control-Allow-Origin设置为请求值
	//
	// 在预请求和普通请求中都设置
	AllowedOrigins []string
	// []bye{} represent true
	// 1. AllowedHeaders==true 会将Access-Control-Allow-Headers 设置为请求的对应header
	// 2. 其它将 Access-Control-Allow-Headers 设置为此字段值
	//
	// 在预请求中设置
	AllowedHeaders []byte
	// 1. 总是设置为此字段中值，当其不为空时
	//
	// 在非预请求中设置
	ExposedHeaders string
	// 1. len(MaxAge)==0，不设置对应响应头
	//
	// 在预请求中设置
	MaxAge []byte
	// 1. 如果此字段为true，会设置对应的响应头
	//
	// 在预请求和之后的请求都会设置
	SupportsCredentials    bool
	AllowedOriginsPatterns []*regexp.Regexp
}

var DefCfg = Config{
	Path:           []string{"/api", "/sanctum/csrf-cookie"},
	AllowedMethods: []byte{'*'},
	AllowedOrigins: []string{"*"},
	AllowedHeaders: []byte{'*'},
}

// New creates a new middleware handler
func New(cfgs ...*Config) routing.Handler {
	var cfg *Config
	if len(cfgs) != 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	} else {
		cfg = &DefCfg
	}
	return func(ctx *routing.Ctx) error {
		if cfg.Skip != nil && cfg.Skip(ctx) {
			return ctx.Next()
		}
		if !cfg.hasMatchingPath(ctx) {
			return ctx.Next()
		}
		if isPreflightRequest(ctx) {
			cfg.handlePreflightRequest(ctx)
			ctx.Response.Header.AddCanonicalNoS(routing.HeaderVary, routing.HeaderAccessControlRequestMethod)
			return nil
		}
		err := ctx.Next()
		if unsafefn.BtoS(ctx.Method()) == routing.MethodOptions {
			ctx.Response.Header.AddCanonicalNoS(routing.HeaderVary, routing.HeaderAccessControlRequestMethod)
		}
		cfg.addActualRequestHeaders(ctx)
		return err
	}
}

func (cfg *Config) addActualRequestHeaders(ctx *routing.Ctx) {
	cfg.configureAllowedOrigin(ctx)
	if len(ctx.Response.Header.Peek(routing.HeaderAccessControlAllowOrigin)) != 0 {
		if cfg.SupportsCredentials {
			ctx.Response.Header.SetVCanonicalNoS(routing.HeaderAccessControlAllowCredentials, routing.TrueBytes)
		}
		if len(cfg.ExposedHeaders) != 0 {
			ctx.Response.Header.SetCanonicalNoS(routing.HeaderAccessControlExposeHeaders, cfg.ExposedHeaders)
		}
	}
}
func (cfg *Config) hasMatchingPath(ctx *routing.Ctx) (match bool) {
	if len(cfg.Path) == 0 {
		return
	}
	if cfg.Path[0] == "*" {
		match = true
		return
	}
	path := unsafefn.BtoS(ctx.Path())
	for i := range cfg.Path {
		if strings.HasPrefix(path, cfg.Path[i]) {
			match = true
			return
		}
	}
	return
}
func isPreflightRequest(ctx *routing.Ctx) (is bool) {
	return unsafefn.BtoS(ctx.Method()) == routing.MethodOptions && len(ctx.Request.Header.Peek(routing.HeaderAccessControlRequestMethod)) != 0
}
func (cfg *Config) handlePreflightRequest(ctx *routing.Ctx) {
	ctx.SetStatusCode(routing.StatusNoContent)
	cfg.addPreflightRequestHeaders(ctx)
}
func (cfg *Config) addPreflightRequestHeaders(ctx *routing.Ctx) {
	cfg.configureAllowedOrigin(ctx)
	if len(ctx.Response.Header.Peek(routing.HeaderAccessControlAllowOrigin)) == 0 {
		return
	}
	if cfg.SupportsCredentials {
		ctx.Response.Header.SetVCanonicalNoS(routing.HeaderAccessControlAllowCredentials, routing.TrueBytes)
	}
	cfg.configureAllowedMethods(ctx)
	cfg.configureAllowedHeaders(ctx)

	if len(cfg.MaxAge) != 0 {
		ctx.Response.Header.SetVCanonicalNoS(routing.HeaderAccessControlMaxAge, cfg.MaxAge)
	}
}
func (cfg *Config) configureAllowedOrigin(ctx *routing.Ctx) {
	if len(cfg.AllowedOrigins) == 0 && cfg.AllowedOrigins != nil && !cfg.SupportsCredentials {
		ctx.Response.Header.SetVCanonicalNoS(routing.HeaderAccessControlAllowOrigin, routing.StartBytes)
		return
	}
	origin := ctx.Request.Header.Peek(routing.HeaderOrigin)
	if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" && len(cfg.AllowedOriginsPatterns) == 0 {
		if origin == nil {
			return
		}
		ctx.Response.Header.SetVCanonicalNoS(routing.HeaderAccessControlAllowOrigin, origin)
		return
	}
	// dynamic
	if len(ctx.Request.Header.Peek(routing.HeaderOrigin)) != 0 && cfg.isOriginAllowed(ctx) {
		if origin != nil {
			ctx.Response.Header.SetVCanonicalNoS(routing.HeaderAccessControlAllowOrigin, origin)
		}
	}
	ctx.Response.Header.AddCanonicalNoS(routing.HeaderVary, routing.HeaderOrigin)
}

func (cfg *Config) isOriginAllowed(ctx *routing.Ctx) (allowed bool) {
	if len(cfg.AllowedOrigins) == 0 && cfg.AllowedOrigins != nil {
		allowed = true
		return
	}
	origin := ctx.Request.Header.Peek(routing.HeaderOrigin)
	if slices.Contains(cfg.AllowedOrigins, unsafefn.BtoS(origin)) {
		allowed = true
		return
	}
	for i := range cfg.AllowedOriginsPatterns {
		if cfg.AllowedOriginsPatterns[i].Match(origin) {
			allowed = true
			return
		}
	}
	return
}
func (cfg *Config) configureAllowedMethods(ctx *routing.Ctx) {
	var allowMethods []byte
	if len(cfg.AllowedMethods) == 0 && cfg.AllowedMethods != nil {
		allowMethods = ctx.Request.Header.Peek(routing.HeaderAccessControlRequestMethod)
		ctx.Response.Header.AddCanonicalNoS(routing.HeaderVary, routing.HeaderAccessControlRequestMethod)
	} else {
		allowMethods = cfg.AllowedMethods
	}
	if len(allowMethods) != 0 {
		ctx.Response.Header.SetVCanonicalNoS(routing.HeaderAccessControlAllowMethods, allowMethods)
	}
}
func (cfg *Config) configureAllowedHeaders(ctx *routing.Ctx) {
	var allowHeaders []byte
	if len(cfg.AllowedHeaders) == 0 && cfg.AllowedHeaders != nil {
		allowHeaders = ctx.Request.Header.Peek(routing.HeaderAccessControlRequestHeaders)
		ctx.Response.Header.AddCanonicalNoS(routing.HeaderVary, routing.HeaderAccessControlRequestHeaders)
	} else {
		allowHeaders = cfg.AllowedHeaders
	}
	if len(allowHeaders) != 0 {
		ctx.Response.Header.SetVCanonicalNoS(routing.HeaderAccessControlAllowHeaders, allowHeaders)
	}
}
