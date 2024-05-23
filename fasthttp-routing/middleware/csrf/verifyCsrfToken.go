package csrf

import (
	"crypto/hmac"
	"errors"
	"strings"
	"time"

	routing "fasthttp-routing"
	"fasthttp-routing/middleware/session"
	"github.com/newacorn/fasthttp"
	"helpers/unsafefn"
)

var mismatchError = errors.New("csrf token mismatch")

const ReadingMethod = "HEADGETOPTIONS"
const HeaderNameFromMeta = "X-CSRF-TOKEN"
const HeaderNameFromCookie = "X-XSRF-TOKEN"
const NameFromFrom = "_token"

// const NameInSession = "_token"

type Config struct {
	skip         routing.Skipper
	except       []string
	noCookie     bool
	cokLifetime  time.Duration
	cokPath      string
	cokDomain    string
	cokHttpOnly  bool
	cokSame_site fasthttp.CookieSameSite
	cokSecure    bool
}

var DefCfg = Config{
	cokLifetime:  time.Hour * 7 * 24,
	cokSame_site: fasthttp.CookieSameSiteNoneMode,
}

func New(cfgs ...*Config) routing.Handler {
	var cfg *Config
	if len(cfgs) > 0 && cfgs[0] != nil {
		// return cfgs[0].handle
		cfg = cfgs[0]
	} else {
		cfg = &DefCfg
	}
	return cfg.handle
}
func (cfg *Config) handle(c *routing.Ctx) (err error) {
	sessionData := c.UserValue(session.ContextKey).(*session.Data)
	token2 := sessionData.CsrfToken()
	if cfg.isReading(c) || cfg.inExceptArray(c) || cfg.tokensMatch(c, token2) || sessionData.Manager().AppendHash.ValidateHash(token2) {
		err = c.Next()
		if cfg.noCookie || len(token2) == 0 {
			return
		}
		cok := cfg.newCookie(token2)
		c.Response.Header.SetCookie(cok)
		fasthttp.ReleaseCookie(cok)
		return
	}
	return mismatchError
}
func (cfg *Config) isReading(c *routing.Ctx) bool {
	return strings.Contains(ReadingMethod, unsafefn.BtoS(c.Method()))
}
func (cfg *Config) inExceptArray(c *routing.Ctx) bool {
	for i := range cfg.except {
		if strings.HasPrefix(unsafefn.BtoS(c.Path()), cfg.except[i]) {
			return true
		}
	}
	return false
}
func (cfg *Config) tokensMatch(c *routing.Ctx, token2 []byte) bool {
	token := getTokenFromRequest(c)

	if len(token) != 0 && len(token2) != 0 {
		return hmac.Equal(token, token2)
	}
	return false
}

func getTokenFromRequest(c *routing.Ctx) (token []byte) {
	token = c.FormValue(NameFromFrom)
	if len(token) == 0 {
		token = c.Request.Header.Peek(HeaderNameFromMeta)
	}
	if len(token) == 0 {
		token = c.Request.Header.Peek(HeaderNameFromCookie)
	}
	// 如果cookie经过加密这里还需要解密处理
	return
}
func (cfg *Config) newCookie(csrfToken []byte) (cok *fasthttp.Cookie) {
	cok = fasthttp.AcquireCookie()
	if cfg.cokDomain != "" {
		cok.SetDomain(cfg.cokDomain)
	}
	if cfg.cokPath != "" {
		cok.SetPath(cfg.cokPath)
	}
	cok.SetKey("XSRF-TOKEN")
	cok.SetValueBytes(csrfToken)
	// cok.SetExpire(time.Now().Add(cfg.cokLifetime * time.Second))
	cok.SetMaxAge(int(cfg.cokLifetime.Seconds()))
	cok.SetSecure(cfg.cokSecure)
	cok.SetSameSite(cfg.cokSame_site)
	return
}
