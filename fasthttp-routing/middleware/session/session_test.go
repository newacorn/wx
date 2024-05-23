package session

import (
	"context"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	routing "fasthttp-routing"
	"github.com/deckarep/golang-set/v2"
	"helpers/unsafefn"
)

type testServer struct {
	*routing.Router
}

func newTestServer(t *testing.T, app *routing.Router) *testServer {
	return &testServer{app}
}

func (ts *testServer) execute(t *testing.T, urlPath string, reqCookie *http.Cookie) (*http.Cookie, string) {
	req := httptest.NewRequest("GET", urlPath, nil)
	if reqCookie != nil {
		req.AddCookie(reqCookie)
	}

	res, err := ts.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	var resCookie *http.Cookie
	if len(res.Cookies()) > 0 {
		resCookie = res.Cookies()[0]
	}

	return resCookie, string(body)
}

func TestEnable(t *testing.T) {
	t.Parallel()
	app := routing.New()
	app.Use(New())

	app.Get("/put", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		d.Put("foo", "bar")
		return nil
	})
	app.Get("/get", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		s := d.Get("foo").(string)
		c.SetBodyString(s)
		return nil
	})

	ts := newTestServer(t, app)

	cookie1, _ := ts.execute(t, "/put", nil)
	token1 := cookie1.Value

	cookie2, body := ts.execute(t, "/get", cookie1)
	if body != "bar" {
		t.Errorf("want %q; got %q", "bar", body)
	}
	if cookie2.Value != cookie1.Value {
		t.Errorf("want %q; got %q", cookie1.Value, cookie2.Value)
	}

	cookie3, _ := ts.execute(t, "/put", cookie1)
	token2 := cookie3.Value
	if token1 != token2 {
		t.Error("want tokens to be the same")
	}
}

func TestLifetime(t *testing.T) {
	t.Parallel()

	app := routing.New()

	// sessionAdapter := New(sessionManager)
	cfg := DefCfg
	cfg.Lifetime = 500 * time.Millisecond
	app.Use(New(&cfg))

	app.Get("/put", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		d.Put("foo", "bar")
		return nil
	})
	app.Get("/get", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		v := d.Get("foo")
		if v == nil {
			c.SetBodyString("foo does not exist in session")
			return nil
		}
		c.SetBodyString(v.(string))
		return nil
	})

	ts := newTestServer(t, app)

	cookie, _ := ts.execute(t, "/put", nil)

	_, body := ts.execute(t, "/get", cookie)
	if body != "bar" {
		t.Errorf("want %q; got %q", "bar", body)
	}
	time.Sleep(time.Second)

	_, body = ts.execute(t, "/get", cookie)
	if body != "foo does not exist in session" {
		t.Errorf("want %q; got %q", "foo does not exist in session", body)
	}
}

func TestLifetimeRefresh(t *testing.T) {
	t.Parallel()

	app := routing.New()
	cfg := DefCfg
	cfg.Lifetime = 4 * time.Second
	app.Use(New(&cfg))

	app.Get("/put", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		d.Put("foo", "bar")
		return nil
	})
	app.Get("/get", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		v := d.Get("foo")
		if v == nil {
			c.SetBodyString("foo does not exist in session")
			return nil
		}
		c.SetBodyString(v.(string))
		return nil
	})

	ts := newTestServer(t, app)

	cookie, _ := ts.execute(t, "/put", nil)

	time.Sleep(2 * time.Second)
	_, _ = ts.execute(t, "/get", cookie)

	time.Sleep(2 * time.Second)
	_, body := ts.execute(t, "/get", cookie)
	if body != "bar" {
		t.Errorf("want %q; got %q", "bar", body)
	}

	time.Sleep(2 * time.Second)
	_, body = ts.execute(t, "/get", cookie)
	if body != "bar" {
		t.Errorf("want %q; got %q", "bar", body)
	}
}

func TestDestroy(t *testing.T) {
	t.Parallel()

	app := routing.New()
	// sessionAdapter := New(sessionManager)
	app.Use(New())

	app.Get("/put", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		d.Put("foo", "bar")
		return nil
	})
	app.Get("/destroy", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		ctx := context.Background()
		err := d.Manager().Destroy(ctx, c)
		if err != nil {
			c.SetStatusCode(500)
			return nil
		}
		return nil
	})
	app.Get("/get", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		v := d.Get("foo")
		if v == nil {
			c.SetBodyString("foo does not exist in session")
			return nil
		}
		c.SetBodyString(v.(string))
		return nil
	})

	ts := newTestServer(t, app)

	cookie1, _ := ts.execute(t, "/put", nil)
	cookie, _ := ts.execute(t, "/destroy", cookie1)

	if cookie1.Value == cookie.Value {
		t.Fatalf("cookie1 should be different from cookie")
	}
	/*
		if strings.HasPrefix(cookie.String(), fmt.Sprintf("%s=;", DefCfg.CokName)) == false {
			t.Fatalf("got %q: expected prefix %q", cookie, fmt.Sprintf("%s=;", DefCfg.CokName))
		}
		if strings.Contains(cookie.String(), "Expires=Tue, 10 Nov 2009 23:00:00 GMT") == false {
			t.Fatalf("got %q: expected to contain %q", cookie, "Expires=Tue, 10 Nov 2009 23:00:00 GMT")
		}
		if strings.Contains(cookie.String(), "Max-Age") {
			t.Fatalf("got %q: expected to contain %q", cookie, "Max-Age=0")
		}
	*/

	_, body := ts.execute(t, "/get", cookie)
	if body != "foo does not exist in session" {
		t.Errorf("want %q; got %q", "foo does not exist in session", body)
	}
}

func TestRenewToken(t *testing.T) {
	t.Parallel()

	app := routing.New()
	app.Use(New())

	app.Get("/put", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		d.Put("foo", "bar")
		return nil
	})
	app.Get("/renew", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		err := d.Manager().RenewToken(context.Background(), c)
		if err != nil {
			c.SetStatusCode(500)
			return nil
		}
		return nil
	})
	app.Get("/get", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		v := d.Get("foo")
		if v == nil {
			c.SetBodyString("foo does not exist in session")
			return nil
		}
		c.SetBodyString(v.(string))
		return nil
	})

	ts := newTestServer(t, app)

	cookie, _ := ts.execute(t, "/put", nil)
	originalToken := cookie.Value

	cookie, _ = ts.execute(t, "/renew", cookie)
	newToken := cookie.Value

	if newToken == originalToken {
		t.Fatal("token has not changed")
	}

	_, body := ts.execute(t, "/get", cookie)
	if body != "bar" {
		t.Errorf("want %q; got %q", "bar", body)
	}
}

func TestRememberMe(t *testing.T) {
	t.Parallel()

	app := routing.New()

	cfg := DefCfg
	cfg.DisableCokPersist = true
	app.Use(New(&cfg))

	app.Get("/put-normal", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		d.Put("foo", "bar")
		return nil
	})
	app.Get("/put-rememberMe-true", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		d.RememberMe(true)
		d.Put("foo", "bar")
		return nil
	})
	app.Get("/put-rememberMe-false", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		d.RememberMe(false)
		d.Put("foo", "bar")
		return nil
	})

	ts := newTestServer(t, app)

	cookie, _ := ts.execute(t, "/put-normal", nil)

	if strings.Contains(cookie.String(), "Max-Age=") || strings.Contains(cookie.String(), "Expires=") {
		t.Errorf("want no Max-Age or Expires attributes; got %q", cookie.String())
	}

	cookie, _ = ts.execute(t, "/put-rememberMe-true", cookie)

	if !strings.Contains(cookie.String(), "Max-Age=") && !strings.Contains(cookie.String(), "Expires=") {
		t.Errorf("want Max-Age or Expires attributes; got %q", cookie.String())
	}

	cookie, _ = ts.execute(t, "/put-rememberMe-false", cookie)

	if strings.Contains(cookie.String(), "Max-Age=") || strings.Contains(cookie.String(), "Expires=") {
		t.Errorf("want no Max-Age or Expires attributes; got %q", cookie.String())
	}
}

func TestIterate(t *testing.T) {
	t.Parallel()

	app := routing.New()
	app.Use(New())
	var SessionManager *Manager

	app.Get("/put", func(c *routing.Ctx) error {
		d, _ := c.UserValue(ContextKey).(*Data)
		SessionManager = d.manager
		d.Put("foo", unsafefn.BtoS(c.QueryArgs().Peek("foo")))
		return nil
	})

	originalResults := [20]string{}
	for i := 0; i < 20; i++ {
		ts := newTestServer(t, app)
		v := rand.Int()
		originalResults[i] = strconv.Itoa(v)
		ts.execute(t, "/put?foo="+strconv.Itoa(v), nil)
	}

	results := mapset.NewSetWithSize[string](20)
	err := SessionManager.Iterate(context.Background(), &routing.Ctx{}, func(ctx context.Context) error {
		d, _ := ctx.Value(ContextKey).(*Data)
		i := d.GetString("foo")
		results.Add(i)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	if !results.Contains(originalResults[:]...) {
		t.Fatalf("unexpected value: got %v; original:%v", results, originalResults)
	}

	err = SessionManager.Iterate(context.Background(), &routing.Ctx{}, func(ctx context.Context) error {
		return errors.New("expected error")
	})
	if err.Error() != "expected error" {
		t.Fatal("didn't get expected error")
	}
}
