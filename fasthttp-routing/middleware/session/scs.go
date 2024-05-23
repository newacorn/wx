package session

import (
	"context"
	"net/http"
	"sync"
	"time"

	routing "fasthttp-routing"
	"fasthttp-routing/middleware/session/redisstore"
	"github.com/newacorn/fasthttp"
	"github.com/redis/rueidis"
	"github.com/rs/zerolog/log"
	"helpers/utilcrypt"
)

type contextKey string

var ContextKey contextKey = "session"

const CsrfTokenLen = 20
const AppendHashCsrfTokenLen = CsrfTokenLen + 2
const UrlEncodedCsrfTokenLen = (AppendHashCsrfTokenLen*8 + 5) / 6

const TokenLen = 30
const AppendHashTokenLen = TokenLen + 2
const UrlEncodedTokenLen = (AppendHashTokenLen*8 + 5) / 6

// UrlEncodedTokenWithPrefixLen comment
const UrlEncodedTokenWithPrefixLen = UrlEncodedTokenLen + len(sessionPrefix)
const sessionPrefix = "scs:session:"

type DataPool struct {
	sync.Pool
}

func (p *DataPool) Acquire() *Data {
	return p.Get().(*Data)
}
func (p *DataPool) Release(data *Data) {
	data.reset()
	p.Put(data)
}

// Manager  holds the configuration settings for your sessions.
type Manager struct {
	// Lifetime controls the maximum length of time that a session is valid for
	// before it expires. The lifetime is an 'absolute expiry' which is set when
	// the session is first created and **does not change**. The default value is 24
	// hours.
	Lifetime time.Duration

	// Store controls the session store where the session data is persisted.
	Store   Store
	CokName string

	// Cookie contains the configuration settings for session cookies.
	// Cookie Cookie

	// Codec controls the encoder/decoder used to transform session data to a
	// byte slice for use by the session store. By default session data is
	// encoded/decoded using encoding/gob.
	Codec Codec

	// ErrorFunc allows you to control behavior when an error is encountered by
	// the LoadAndSave middleware. The default behavior is for a HTTP 500
	// "Internal Server Error" message to be sent to the client and the error
	// logged using Go's standard logger. If a custom ErrorFunc is set, then
	// control will be passed to this instead. A typical use would be to provide
	// a function which logs the error and returns a customized HTML error page.
	ErrorFunc  func(ctx *routing.Ctx, err error)
	IdSize     int
	AppendHash utilcrypt.BytesWithHash
	dataPool   DataPool

	// contextKey is the key used to set and retrieve the session data from a
	// context.Context. It's automatically generated to ensure uniqueness.
	contextKey contextKey
}

// Cookie  contains the configuration settings for session cookies.
type Cookie struct {
	// Name sets the name of the session cookie. It should not contain
	// whitespace, commas, colons, semicolons, backslashes, the equals sign or
	// control characters as per RFC6265. The default cookie name is "session".
	// If your application uses two different sessions, you must make sure that
	// the cookie name for each is unique.
	Name string

	// Domain sets the 'Domain' attribute on the session cookie. By default
	// it will be set to the domain name that the cookie was issued from.
	Domain string

	// HttpOnly sets the 'HttpOnly' attribute on the session cookie. The
	// default value is true.
	HttpOnly bool

	// Path sets the 'Path' attribute on the session cookie. The default value
	// is "/". Passing the empty string "" will result in it being set to the
	// path that the cookie was issued from.
	Path string

	// Persist sets whether the session cookie should be persistent or not
	// (i.e. whether it should be retained after a user closes their browser).
	// The default value is true, which means that the session cookie will not
	// be destroyed when the user closes their browser and the appropriate
	// 'Expires' and 'MaxAge' values will be added to the session cookie. If you
	// want to only persist some sessions (rather than all of them), then set this
	// to false and call the RememberMe() method for the specific sessions that you
	// want to persist.
	Persist bool

	// SameSite controls the value of the 'SameSite' attribute on the session
	// cookie. By default this is set to 'SameSite=Lax'. If you want no SameSite
	// attribute or value in the session cookie then you should set this to 0.
	SameSite http.SameSite

	// Secure sets the 'Secure' attribute on the session cookie. The default
	// value is false. It's recommended that you set this to true and serve all
	// requests over HTTPS in production environments.
	// See https://github.com/OWASP/CheatSheetSeries/blob/master/cheatsheets/Session_Management_Cheat_Sheet.md#transport-layer-security.
	Secure bool
}
type Config struct {
	manager   *Manager
	Skip      routing.Skipper
	Lifetime  time.Duration
	Store     Store
	Codec     Codec
	ErrorFunc func(ctx *routing.Ctx, err error)
	//
	CokName                string
	CokDomain              string
	CokPath                string
	CokSameSite            http.SameSite
	DisableCokPersist      bool
	DisableCokHttpHttpOnly bool
	CokSecure              bool
	disableCsrf            bool
	//
	IdSize     int
	AppendHash utilcrypt.BytesWithHash
}

var DefCfg = Config{
	Lifetime:    time.Hour * 24 * 7,
	IdSize:      30,
	CokName:     "new_fire_session",
	Codec:       JSONCodec{},
	CokSameSite: http.SameSiteLaxMode,
	AppendHash:  utilcrypt.SimpleHash{},
}

func New(cfgs ...*Config) routing.Handler {
	var cfg *Config
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	} else {
		cfg = &DefCfg
	}
	cfg.initSessionManager()
	return cfg.Handle
}

// Handle 此中间件可能返回的错误:
// EncodingError (在序列化时遇到错误)
// StoreError (在从Store中加载数据或者存储数据时遇到错误)
// 当遇到这两种错误时，此中间件不会像客户端发送session cookie
func (cfg *Config) Handle(c *routing.Ctx) (err error) {
	if cfg.Skip != nil && cfg.Skip(c) {
		return c.Next()
	}
	// ctx := context.Background()
	ctx := context.Context(nil)
	data, newToken := cfg.getSession(c)
	err = cfg.handleStatefulRequest(c, ctx, data, newToken)
	cfg.manager.dataPool.Release(data)
	return
}

// 可能返回的错误:
// EncodingError (在序列化时遇到错误)
// StoreError (在从Store中加载数据或者存储数据时遇到错误)
func (cfg *Config) handleStatefulRequest(c *routing.Ctx, ctx context.Context, data *Data, newToken bool) (err error) {
	err = data.Start(newToken, ctx)
	if err != nil {
		return
	}
	c.SetUserValue(ContextKey, data)
	err = c.Next()
	err2 := cfg.manager.commit(ctx, c, data)
	//
	if err == nil {
		err = err2
	}
	if err2 != nil {
		log.Error().Str("Error", err2.Error()).Msg("sessionManager commit data occur error")
		return
	}
	if data.status == Destroyed {
		c.Response.Header.DelClientCookie(cfg.CokName)
		return
	}
	cfg.addCookieToResponse(c, data.token, data.GetBool("__rememberMe"))
	return
}
func (cfg *Config) addCookieToResponse(c *routing.Ctx, token []byte, rememberMe bool) {
	cok := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cok)
	cok.SetKey(cfg.CokName)
	cok.SetValueBytes(token)
	if cfg.CokPath != "" {
		cok.SetPath(cfg.CokPath)
	}
	if cfg.CokDomain != "" {
		cok.SetDomain(cfg.CokDomain)
	}
	if cfg.CokSecure {
		cok.SetSecure(true)
	}
	if cfg.DisableCokHttpHttpOnly {
		cok.SetHTTPOnly(false)
	}
	cok.SetSameSite(fasthttp.CookieSameSite(cfg.CokSameSite))

	if !cfg.DisableCokPersist || rememberMe {
		cok.SetMaxAge(int(cfg.Lifetime.Seconds())) // Round up to the nearest second.
	}
	c.Response.Header.SetCookie(cok)
	// c.Response.Header.Add("Cache-Control", `no-cache="Set-Cookie"`)
}
func (cfg *Config) getSession(c *routing.Ctx) (data *Data, newToken bool) {
	data = cfg.manager.dataPool.Acquire()
	token := c.Request.Header.Cookie(cfg.CokName)
	newToken = data.SetToken(token)
	return
}
func (cfg *Config) initSessionManager() {
	if cfg.Store == nil {
		c, err := rueidis.NewClient(rueidis.ClientOption{
			ForceSingleClient: true,
			InitAddress:       []string{"127.0.0.1:6379"},
		})
		if err != nil {
			log.Fatal().Caller().Err(err).Send()
		}
		cfg.Store = redisstore.New(c)
	}
	m := new(Manager)
	m.AppendHash = cfg.AppendHash
	m.IdSize = cfg.IdSize
	m.Store = cfg.Store
	m.CokName = cfg.CokName
	m.Codec = cfg.Codec
	m.Lifetime = cfg.Lifetime
	m.ErrorFunc = cfg.ErrorFunc
	// encodedTokenLen := base64.RawURLEncoding.EncodedLen(m.AppendHash.EncodedLen(cfg.IdSize))
	// encodedCsrfTokenLen := base64.RawURLEncoding.EncodedLen(m.AppendHash.EncodedLen(CsrfTokenLen))
	m.dataPool = DataPool{sync.Pool{
		New: func() any {
			data := &Data{}
			// data.token = make([]byte, 0, encodedTokenLen)
			// data.token = make([]byte, 0, UrlEncodedTokenLen)
			// data.tokenP = make([]byte, 0, UrlEncodedTokenWithPrefixLen)
			tokenWithPrefix := make([]byte, UrlEncodedTokenWithPrefixLen)
			copy(tokenWithPrefix, sessionPrefix)
			data.token = tokenWithPrefix[len(sessionPrefix):len(sessionPrefix)]
			if !cfg.disableCsrf {
				data.csrfToken = make([]byte, 0, UrlEncodedCsrfTokenLen)
				// data.csrfToken = make([]byte, 0, encodedCsrfTokenLen)
			}
			data.values = make(map[string]interface{})
			// data.deadline = time.Now().Add(cfg.Lifetime).UTC()
			data.manager = m
			return data
		},
	}}
	cfg.manager = m
}
