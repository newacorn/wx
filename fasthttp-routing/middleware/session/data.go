package session

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"
	"unsafe"

	routing "fasthttp-routing"
	"github.com/rs/zerolog/log"
	"helpers/unsafefn"
	"helpers/utilcrypt"
)

// Status represents the state of the session data during a request cycle.
type Status int

const (
	// Unmodified indicates that the session data hasn't been changed in the
	// current request cycle.
	Unmodified Status = iota

	// Modified indicates that the session data has been changed in the current
	// request cycle.
	Modified

	// Destroyed indicates that the session data has been destroyed in the
	// current request cycle.
	Destroyed
)

type Data struct {
	status    Status
	token     []byte
	csrfToken []byte
	values    map[string]interface{}
	manager   *Manager
	started   bool
}

func (d *Data) reset() {
	d.token = d.token[:0]
	d.csrfToken = d.csrfToken[:0]
	clear(d.values)
	d.status = Unmodified
	d.started = false
}

func (d *Data) Manager() *Manager {
	return d.manager
}

func GenerateToken2(dstRaw []byte, dst []byte, appendHash utilcrypt.BytesWithHash) (token []byte) {

	_, err := unsafefn.NoescapeRead(rand.Reader, dstRaw[:len(dstRaw)-2])
	if err != nil {
		err = NewTokenError("genToken:" + err.Error())
		log.Panic().Str("Err", err.Error()).Msg("gen session token")
		return
	}
	utilcrypt.NoescapeAppendHash(appendHash, dstRaw)
	// appendHash.AppendHash(dstRaw)
	base64.RawURLEncoding.Encode(dst, dstRaw)
	token = dst
	return
}

func GenerateToken(size int, dst []byte, appendHash utilcrypt.BytesWithHash, urlencoded bool) (token []byte) {

	var urlEncodedLen, rawTokenLen int
	var rawToken []byte

	if appendHash != nil {
		rawTokenLen = appendHash.EncodedLen(size)
		// rawToken = make([]byte, appendHashEncodedLen)
	} else {
		rawTokenLen = size
	}
	if !urlencoded && cap(dst) >= rawTokenLen {
		rawToken = dst[:rawTokenLen]
	} else {
		rawToken = make([]byte, rawTokenLen)
	}
	_, err := rand.Read(rawToken[:size])
	if err != nil {
		err = NewTokenError("genToken:" + err.Error())
		log.Panic().Str("Err", err.Error()).Msg("gen session token")
		return
	}
	if appendHash != nil {
		appendHash.AppendHash(rawToken)
	}
	if urlencoded {
		urlEncodedLen = base64.RawURLEncoding.EncodedLen(len(rawToken))
		if cap(dst) >= urlEncodedLen {
			token = dst[:urlEncodedLen]
		} else {
			token = make([]byte, urlEncodedLen)
		}
		base64.RawURLEncoding.Encode(token, rawToken)
		return
	}
	token = rawToken
	return
}

func (d *Data) Put(key string, val any) {
	d.values[key] = val
	d.status = Modified
}
func (d *Data) SetToken(token []byte) (newToken bool) {
	if len(token) > 0 && d.isValidToken(token, d.manager.AppendHash) {
		d.token = append(d.token[:0], token...)
		return
	}
	// d.token = GenerateToken(d.manager.IdSize, d.token[:0], routing.SimpleHash{}, true)
	dstRaw := make([]byte, AppendHashTokenLen)
	d.token = GenerateToken2(dstRaw, d.token[:UrlEncodedTokenLen], d.manager.AppendHash)
	return true
}
func (d *Data) Start(newToken bool, ctx context.Context) (err error) {
	if !newToken {
		err = d.LoadSession(ctx)
		if err != nil {
			return
		}
	}

	if len(d.csrfToken) == 0 {
		dstRaw := make([]byte, AppendHashCsrfTokenLen)
		d.csrfToken = GenerateToken2(dstRaw, d.csrfToken[:UrlEncodedCsrfTokenLen], d.manager.AppendHash)
		// d.RegenerateCsrfToken()
	}
	d.started = true
	return
}
func (d *Data) LoadSession(ctx context.Context) (err error) {
	// 根据token从存储中加载序列化的session数据，并解码到sessionData类型实例
	dataBytes, found, err := d.manager.doStoreFind(ctx, (*routing.Ctx)(nil), d.token)
	if err != nil {
		err = NewStoreError("find session data from session store: " + err.Error())
		// 从存储中索取数据时遇到错误，这类错误不生成新的Session
		return
	}
	if !found {
		return
	}
	if d.csrfToken, d.values, err = d.manager.Codec.Decode(dataBytes, d.csrfToken[0:0]); err != nil {
		// 数据损坏，解码错误，删除对应的数据
		if err != nil {
			log.Warn().Str("Err", err.Error()).Msg("decode session data occur error")
		}
		// 删除损坏数据以免无限循环
		tokenWithPrefix := unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(d.token)), -len(sessionPrefix))), UrlEncodedTokenWithPrefixLen)

		err = d.manager.Store.Delete(tokenWithPrefix)
		if err != nil {
			log.Warn().Str("Error", err.Error()).Msg("session data corrupt->delete token from session store")
		}
		err = nil
		return
	}
	return
}
func (d *Data) isValidToken(token []byte, appendHash utilcrypt.BytesWithHash) bool {
	// token存在与cookie中
	urlDecodedToken := make([]byte, base64.RawURLEncoding.DecodedLen(len(token)))
	n, err := base64.RawURLEncoding.Decode(urlDecodedToken, token)
	urlDecodedToken = urlDecodedToken[:n]
	if err != nil || !appendHash.ValidateHash(urlDecodedToken) {
		if err != nil {
			log.Info().Str("Err", err.Error()).Bytes("token", token).Msg("decode session token occur error")
		} else {
			log.Info().Bytes("token", token).Msg("validate session token hash failure")
		}
		// 来自客户端的token无效,生成一个新的sessionData
		// 忽略这类错误
		return false
	}
	return true
}
func (d *Data) MergeSession(ctx context.Context, c *routing.Ctx, token []byte) (err error) {
	if bytes.Equal(d.token, token) {
		return nil
	}
	var encodedData []byte
	var found bool

	encodedData, found, err = d.manager.doStoreFind(ctx, c, token)
	if err != nil || !found {
		return nil
	}
	_, values, err := d.manager.Codec.Decode(encodedData, nil)
	if err != nil {
		return
	}

	for k, v := range values {
		// 新的同名数据优先于旧的
		_, ok := d.values[k]
		if !ok {
			d.values[k] = v
		}
	}
	d.status = Modified
	return d.manager.doStoreDelete(ctx, token)
}

func (d *Data) Get(key string) interface{} {
	return d.values[key]
}
func (d *Data) CsrfToken() []byte {
	return d.csrfToken
}

func (d *Data) Pop(key string) interface{} {
	val, exists := d.values[key]
	if !exists {
		return nil
	}
	delete(d.values, key)
	d.status = Modified
	return val
}

func (d *Data) Remove(key string) {
	_, exists := d.values[key]
	if !exists {
		return
	}
	delete(d.values, key)
	d.status = Modified
}

func (d *Data) Clear() error {
	if len(d.values) == 0 {
		return nil
	}
	clear(d.values)
	d.status = Modified
	return nil
}

func (d *Data) Exists(key string) bool {
	_, exists := d.values[key]
	return exists
}

func (d *Data) Keys() []string {
	keys := make([]string, len(d.values))
	i := 0
	for key := range d.values {
		keys[i] = key
		i++
	}
	return keys
}

func (d *Data) Flush() {
	clear(d.values)
}

//goland:noinspection GoDirectComparisonOfErrors
func (d *Data) Migrate(destroy bool) {
	if destroy {
		if len(d.token) != 0 {
			tokenWithPrefix := unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(d.token)), -len(sessionPrefix))), UrlEncodedTokenWithPrefixLen)
			err := d.manager.Store.Delete(tokenWithPrefix)
			if err != nil {
				log.Error().Str("Err", err.Error()).Msg("session migrate error on delete token from store")
			}
		}
	}
	d.token = d.GenerateSessionId()
	d.status = Modified
}
func (d *Data) GenerateSessionId() []byte {
	dstRaw := make([]byte, AppendHashTokenLen)
	return GenerateToken2(dstRaw, d.token[:UrlEncodedTokenLen], d.manager.AppendHash)
	// return GenerateToken(d.manager.IdSize, d.token[:0], d.manager.AppendHash, true)
}
func (d *Data) Regenerate(destroy bool) (err error) {
	d.Migrate(destroy)
	d.RegenerateCsrfToken()
	return
}
func (d *Data) RegenerateCsrfToken() {
	// csrfToken, _ := d.values["_token"].([]byte)
	// d.csrfToken = GenerateToken(20, d.csrfToken[:0], d.manager.AppendHash, true)
	dstRaw := make([]byte, CsrfTokenLen)
	d.csrfToken = GenerateToken2(dstRaw, d.csrfToken[:0], d.manager.AppendHash)
	return
}
func (d *Data) Invalidate() {
	d.Flush()
	d.Migrate(true)
}

func (d *Data) Status() Status {
	return d.status
}

func (d *Data) GetString(key string) string {
	val := d.Get(key)
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return str
}

func (d *Data) GetBool(key string) bool {
	val := d.Get(key)
	b, ok := val.(bool)
	if !ok {
		return false
	}
	return b
}

func (d *Data) GetInt(key string) int {
	val := d.Get(key)
	i, ok := val.(int)
	if !ok {
		return 0
	}
	return i
}

func (d *Data) GetInt64(key string) int64 {
	val := d.Get(key)
	i, ok := val.(int64)
	if !ok {
		return 0
	}
	return i
}

func (d *Data) GetInt32(key string) int32 {
	val := d.Get(key)
	i, ok := val.(int32)
	if !ok {
		return 0
	}
	return i
}

func (d *Data) GetFloat(key string) float64 {
	val := d.Get(key)
	f, ok := val.(float64)
	if !ok {
		return 0
	}
	return f
}

func (d *Data) GetTime(key string) time.Time {
	val := d.Get(key)
	t, ok := val.(time.Time)
	if !ok {
		return time.Time{}
	}
	return t
}

func (d *Data) PopString(key string) string {
	val := d.Pop(key)
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return str
}

func (d *Data) PopBool(key string) bool {
	val := d.Pop(key)
	b, ok := val.(bool)
	if !ok {
		return false
	}
	return b

}

func (d *Data) PopInt(key string) int {
	val := d.Pop(key)
	i, ok := val.(int)
	if !ok {
		return 0
	}
	return i
}

func (d *Data) PopFloat(key string) float64 {
	val := d.Pop(key)
	f, ok := val.(float64)
	if !ok {
		return 0
	}
	return f
}

func (d *Data) PopBytes(key string) []byte {
	val := d.Pop(key)
	b, ok := val.([]byte)
	if !ok {
		return nil
	}
	return b
}

func (d *Data) PopTime(key string) time.Time {
	val := d.Pop(key)
	t, ok := val.(time.Time)
	if !ok {
		return time.Time{}
	}
	return t
}

func (d *Data) RememberMe(val bool) {
	d.Put("__rememberMe", val)
}

func (d *Data) Token() (token []byte) {
	return d.token
}
