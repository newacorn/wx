package session

import (
	"context"

	routing "fasthttp-routing"
)

func (s *Manager) commit(ctx context.Context,
	c *routing.Ctx, data *Data) (err error) {
	err = s.Commit(ctx, c, data)
	return
}

// WriteSessionCookie writes a cookie to the HTTP response with the provided
// token as the cookie value and expiry as the cookie expiry time. The expiry
// time will be included in the cookie only if the session is set to persist
// or has had RememberMe(true) called on it. If expiry is an empty time.Time
// struct (so that it's IsZero() method returns true) the cookie will be
// marked with a historical expiry time and negative max-age (so the browser
// deletes it).
//
// Most applications will use the LoadAndSave() middleware and will not need to
// use this method.
