package session

import (
	"context"
	"time"
	"unsafe"

	routing "fasthttp-routing"
	"github.com/rs/zerolog/log"
)

func (s *Manager) addSessionDataToContext(c *routing.Ctx, sd *Data) {
	c.SetUserValue(ContextKey, sd)
}
func (s *Manager) doStoreFind(ctx context.Context, _ *routing.Ctx, token []byte) (b []byte, found bool, err error) {
	tokenWithPrefix := unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(token)), -len(sessionPrefix))), UrlEncodedTokenWithPrefixLen)

	if ctx != nil {
		store1, ok := s.Store.(CtxStore)
		if ok {
			b, found, err = store1.FindCtx(ctx, tokenWithPrefix)
		} else {
			b, found, err = s.Store.Find(tokenWithPrefix)
		}
	} else {
		b, found, err = s.Store.Find(tokenWithPrefix)
	}
	if err != nil {
		err = NewStoreError("find session data:" + err.Error())
		log.Error().Str("Err", err.Error()).Bytes("token", token).Msg("find session data occur error")
	}
	return
}

// Commit saves the session data to the session store and returns the session
// token and expiry time.
//
// Most applications will use the LoadAndSave() middleware and will not need to
// use this method.
func (s *Manager) Commit(ctx context.Context, _ *routing.Ctx, data *Data) (err error) {
	var serializedData []byte
	serializedData, err = s.Codec.Encode(data.csrfToken, data.values)
	if err != nil {
		err = NewEncodingError(err.Error())
		return
	}
	err = s.doStoreCommit(ctx, data.token, serializedData, time.Now().Add(s.Lifetime), data.status == Modified)
	return
}
func (s *Manager) sessionDataFromCxt(c *routing.Ctx) (data *Data) {
	var ok bool
	data, ok = c.UserValue(ContextKey).(*Data)
	if !ok {
		panic("scs: no session data in context")
	}
	return
}

func (s *Manager) generateToken() (token []byte) {
	return GenerateToken(s.IdSize, nil, s.AppendHash, true)
}
func (s *Manager) doStoreCommit(ctx context.Context, token []byte, data []byte, expiry time.Time, modified bool) (err error) {
	tokenWithPrefix := unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(token)), -len(sessionPrefix))), UrlEncodedTokenWithPrefixLen)
	if ctx != nil {
		s1, ok := s.Store.(CtxStore)
		if ok {
			err = s1.CommitCtx(ctx, tokenWithPrefix, data, expiry, modified)
		} else {
			err = s.Store.Commit(tokenWithPrefix, data, expiry, modified)
		}
	} else {
		err = s.Store.Commit(tokenWithPrefix, data, expiry, modified)
	}
	if err != nil {
		err = NewStoreError("commit session data:" + err.Error())
		log.Error().Str("Err", err.Error()).Msg("commit session data occur error")
	}
	return
}

// Destroy deletes the session data from the session store and sets the session
// status to Destroyed. Any further operations in the same request cycle will
// result in a new session being created.
func (s *Manager) Destroy(ctx context.Context, c *routing.Ctx) error {
	sd := s.sessionDataFromCxt(c)
	err := s.doStoreDelete(ctx, sd.token)
	if err != nil {
		return err
	}
	sd.status = Destroyed
	sd.reset()
	return nil
}

// Put adds a key and corresponding value to the session data. Any existing
// value for the key will be replaced. The session data status will be set to
// Modified.
func (s *Manager) Put(c *routing.Ctx, key string, val interface{}) {
	sd := s.sessionDataFromCxt(c)
	sd.Put(key, val)
}

// Get returns the value for a given key from the session data. The return
// value has the type interface{} so will usually need to be type asserted
// before you can use it. For example:
//
//	foo, ok := session.Get(r, "foo").(string)
//	if !ok {
//		return errors.New("type assertion to string failed")
//	}
//
// Also see the GetString(), GetInt(), GetBytes() and other helper methods which
// wrap the type conversion for common types.
func (s *Manager) Get(c *routing.Ctx, key string) interface{} {
	sd := s.sessionDataFromCxt(c)
	return sd.Get(key)
}
func (s *Manager) GetName() string {
	return s.CokName
}
func (s *Manager) SetName(name string) {
	s.CokName = name
}

// Pop acts like a one-time Get. It returns the value for a given key from the
// session data and deletes the key and value from the session data. The
// session data status will be set to Modified. The return value has the type
// interface{} so will usually need to be type asserted before you can use it.
func (s *Manager) Pop(c *routing.Ctx, key string) interface{} {
	sd := s.sessionDataFromCxt(c)
	return sd.Pop(key)
}

// Remove deletes the given key and corresponding value from the session data.
// The session data status will be set to Modified. If the key is not present
// this operation is a no-op.
func (s *Manager) Remove(c *routing.Ctx, key string) {
	sd := s.sessionDataFromCxt(c)
	sd.Remove(key)
}

// Clear removes all data for the current session. The session token and
// lifetime are unaffected. If there is no data in the current session this is
// a no-op.
func (s *Manager) Clear(c *routing.Ctx) error {
	sd := s.sessionDataFromCxt(c)
	return sd.Clear()
}

// Exists returns true if the given key is present in the session data.
func (s *Manager) Exists(c *routing.Ctx, key string) bool {
	sd := s.sessionDataFromCxt(c)
	return sd.Exists(key)
}

// Keys returns a slice of all key names present in the session data, sorted
// alphabetically. If the data contains no data then an empty slice will be
// returned.
func (s *Manager) Keys(c *routing.Ctx) []string {
	sd := s.sessionDataFromCxt(c)
	return sd.Keys()
}

// RenewToken updates the session data to have a new session token while
// retaining the current session data. The session lifetime is also reset and
// the session data status will be set to Modified.
//
// The old session token and accompanying data are deleted from the session store.
//
// To mitigate the risk of session fixation attacks, it's important that you call
// RenewToken before making any changes to privilege levels (e.g. login and
// logout operations). See https://github.com/OWASP/CheatSheetSeries/blob/master/cheatsheets/Session_Management_Cheat_Sheet.md#renew-the-session-id-after-any-privilege-level-change
// for additional information.
func (s *Manager) RenewToken(_ context.Context, c *routing.Ctx) error {
	sd := s.sessionDataFromCxt(c)
	sd.Migrate(true)
	return nil
}

// MergeSession is used to merge in data from a different session in case strict
// session tokens are lost across an oauth or similar redirect flows. Use Clear()
// if no values of the new session are to be used.
func (s *Manager) MergeSession(ctx context.Context, c *routing.Ctx, token []byte) error {
	data := s.sessionDataFromCxt(c)
	return data.MergeSession(ctx, c, token)
}

// Status returns the current status of the session data.
func (s *Manager) Status(c *routing.Ctx) Status {
	sd := s.sessionDataFromCxt(c)
	return sd.Status()
}

// GetString returns the string value for a given key from the session data.
// The zero value for a string ("") is returned if the key does not exist or the
// value could not be type asserted to a string.
func (s *Manager) GetString(c *routing.Ctx, key string) string {
	sd := s.sessionDataFromCxt(c)
	return sd.GetString(key)
}

// GetBool returns the bool value for a given key from the session data. The
// zero value for a bool (false) is returned if the key does not exist or the
// value could not be type asserted to a bool.
func (s *Manager) GetBool(c *routing.Ctx, key string) bool {
	sd := s.sessionDataFromCxt(c)
	return sd.GetBool(key)
}

// GetInt returns the int value for a given key from the session data. The
// zero value for an int (0) is returned if the key does not exist or the
// value could not be type asserted to an int.
func (s *Manager) GetInt(c *routing.Ctx, key string) int {
	sd := s.sessionDataFromCxt(c)
	return sd.GetInt(key)

}

// GetInt64 returns the int64 value for a given key from the session data. The
// zero value for an int64 (0) is returned if the key does not exist or the
// value could not be type asserted to an int64.
func (s *Manager) GetInt64(c *routing.Ctx, key string) int64 {
	sd := s.sessionDataFromCxt(c)
	return sd.GetInt64(key)
}

// GetInt32 returns the int value for a given key from the session data. The
// zero value for an int32 (0) is returned if the key does not exist or the
// value could not be type asserted to an int32.
func (s *Manager) GetInt32(c *routing.Ctx, key string) int32 {
	sd := s.sessionDataFromCxt(c)
	return sd.GetInt32(key)
}

// GetFloat returns the float64 value for a given key from the session data. The
// zero value for an float64 (0) is returned if the key does not exist or the
// value could not be type asserted to a float64.
func (s *Manager) GetFloat(c *routing.Ctx, key string) float64 {
	sd := s.sessionDataFromCxt(c)
	return sd.GetFloat(key)

}

// GetTime returns the time.Time value for a given key from the session data. The
// zero value for a time.Time object is returned if the key does not exist or the
// value could not be type asserted to a time.Time. This can be tested with the
// time.IsZero() method.
func (s *Manager) GetTime(c *routing.Ctx, key string) time.Time {
	sd := s.sessionDataFromCxt(c)
	return sd.GetTime(key)

}

// PopString returns the string value for a given key and then deletes it from the
// session data. The session data status will be set to Modified. The zero
// value for a string ("") is returned if the key does not exist or the value
// could not be type asserted to a string.
func (s *Manager) PopString(c *routing.Ctx, key string) string {
	sd := s.sessionDataFromCxt(c)
	return sd.PopString(key)

}

// PopBool returns the bool value for a given key and then deletes it from the
// session data. The session data status will be set to Modified. The zero
// value for a bool (false) is returned if the key does not exist or the value
// could not be type asserted to a bool.
func (s *Manager) PopBool(c *routing.Ctx, key string) bool {
	sd := s.sessionDataFromCxt(c)
	return sd.PopBool(key)
}

// PopInt returns the int value for a given key and then deletes it from the
// session data. The session data status will be set to Modified. The zero
// value for an int (0) is returned if the key does not exist or the value could
// not be type asserted to an int.
func (s *Manager) PopInt(c *routing.Ctx, key string) int {
	sd := s.sessionDataFromCxt(c)
	return sd.PopInt(key)
}

// PopFloat returns the float64 value for a given key and then deletes it from the
// session data. The session data status will be set to Modified. The zero
// value for an float64 (0) is returned if the key does not exist or the value
// could not be type asserted to a float64.
func (s *Manager) PopFloat(c *routing.Ctx, key string) float64 {
	sd := s.sessionDataFromCxt(c)
	return sd.PopFloat(key)
}

// PopBytes returns the byte slice ([]byte) value for a given key and then
// deletes it from the from the session data. The session data status will be
// set to Modified. The zero value for a slice (nil) is returned if the key does
// not exist or could not be type asserted to []byte.
func (s *Manager) PopBytes(c *routing.Ctx, key string) []byte {
	sd := s.sessionDataFromCxt(c)
	return sd.PopBytes(key)
}

// PopTime returns the time.Time value for a given key and then deletes it from
// the session data. The session data status will be set to Modified. The zero
// value for a time.Time object is returned if the key does not exist or the
// value could not be type asserted to a time.Time.
func (s *Manager) PopTime(c *routing.Ctx, key string) time.Time {
	sd := s.sessionDataFromCxt(c)
	return sd.PopTime(key)
}

// RememberMe controls whether the session cookie is persistent (i.e  whether it
// is retained after a user closes their browser). RememberMe only has an effect
// if you have set SessionManager.Cookie.Persist = false (the default is true) and
// you are using the standard LoadAndSave() middleware.
func (s *Manager) RememberMe(c *routing.Ctx, val bool) {
	sd := s.sessionDataFromCxt(c)
	sd.RememberMe(val)
}

// Iterate retrieves all active (i.e. not expired) sessions from the store and
// executes the provided function fn for each session. If the session store
// being used does not support iteration then Iterate will panic.
func (s *Manager) Iterate(ctx context.Context, _ *routing.Ctx, fn func(ctx context.Context) error) error {
	allSessions, err := s.doStoreAll(ctx)
	if err != nil {
		return err
	}

	data := s.dataPool.Acquire()
	defer func() {
		s.dataPool.Release(data)
	}()
	for token, b := range allSessions {
		data.token = append(data.token, token...)
		data.manager = s
		data.csrfToken, data.values, err = s.Codec.Decode(b, data.csrfToken[0:0])
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, ContextKey, data)
		err = fn(ctx)
		data.reset()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Manager) doStoreDelete(ctx context.Context, token []byte) (err error) {
	tokenWithPrefix := unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(token)), -len(sessionPrefix))), UrlEncodedTokenWithPrefixLen)
	if ctx != nil {
		s1, ok := s.Store.(CtxStore)
		if ok {
			err = s1.DeleteCtx(ctx, tokenWithPrefix)
			if err != nil {
				err = NewStoreError("Delete:" + err.Error())
				log.Warn().Str("Err", err.Error()).Msg("delete session token from session store")
			}
			return
		}
	}
	err = s.Store.Delete(tokenWithPrefix)
	if err != nil {
		err = NewStoreError("Delete:" + err.Error())
		log.Warn().Str("Err", err.Error()).Msg("delete session token from session store")
	}
	return
}
func (s *Manager) doStoreAll(ctx context.Context) (all map[string][]byte, err error) {
	if ctx != nil {
		cs, ok := s.Store.(IterableCtxStore)
		if ok {
			all, err = cs.AllCtx(ctx)
			if err != nil {
				err = NewStoreError("iterate sessions:" + err.Error())
				log.Warn().Str("Err", err.Error()).Msg("iterate sessions occur error")
			}
			return
		}
	}

	it, ok := s.Store.(IterableStore)
	if ok {
		all, err = it.All()
		if err != nil {
			err = NewStoreError("Delete:" + err.Error())
			log.Warn().Str("Err", err.Error()).Msg("delete session occur error")
		}
		return
	}
	panic("never occur in doStoreAll")
}
