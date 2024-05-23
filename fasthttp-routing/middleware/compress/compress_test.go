package compress

import (
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	routing "fasthttp-routing"
	"github.com/newacorn/fasthttp"
	"github.com/stretchr/testify/assert"
)

var filedata []byte

func init() {
	dat, err := os.ReadFile("../../.github/README.md")
	if err != nil {
		panic(err)
	}
	filedata = dat
}

// go test -run Test_Compress_Gzip
func Test_Compress_Gzip(t *testing.T) {
	t.Parallel()
	r := routing.New()

	r.Use(New())

	r.Get("/", func(c *routing.Ctx) error {
		c.Set(routing.HeaderContentType, routing.MIMETextPlainCharsetUTF8)
		c.Response.SetBodyRaw(filedata)
		return nil
	})

	req := httptest.NewRequest(fasthttp.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := r.Test(req)
	assert.Equal(t, nil, err, "app.Test(req)")
	assert.Equal(t, 200, resp.StatusCode, "Status code")
	assert.Equal(t, "gzip", resp.Header.Get(routing.HeaderContentEncoding))

	// Validate that the file size has shrunk
	body, err := io.ReadAll(resp.Body)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, len(body) < len(filedata))
}

// go test -run Test_Compress_Different_Level
func Test_Compress_Different_Level(t *testing.T) {
	t.Parallel()
	levels := []Level{LevelBestSpeed, LevelBestCompression}
	for _, level := range levels {
		level := level
		t.Run(fmt.Sprintf("level %d", level), func(t *testing.T) {
			t.Parallel()
			r := routing.New()

			r.Use(New(Config{Level: level}))

			r.Get("/", func(c *routing.Ctx) error {
				c.Set(routing.HeaderContentType, routing.MIMETextPlainCharsetUTF8)
				c.Response.SetBodyRaw(filedata)
				return nil
			})

			req := httptest.NewRequest(routing.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			resp, err := r.Test(req)
			assert.Equal(t, nil, err, "app.Test(req)")
			assert.Equal(t, 200, resp.StatusCode, "Status code")
			assert.Equal(t, "gzip", resp.Header.Get(routing.HeaderContentEncoding))

			// Validate that the file size has shrunk
			body, err := io.ReadAll(resp.Body)
			assert.Equal(t, nil, err)
			assert.Equal(t, true, len(body) < len(filedata))
		})
	}
}

func Test_Compress_Deflate(t *testing.T) {
	t.Parallel()
	r := routing.New()

	r.Use(New())

	r.Get("/", func(c *routing.Ctx) error {
		c.Response.SetBodyRaw(filedata)
		return nil
	})

	req := httptest.NewRequest(routing.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "deflate")

	resp, err := r.Test(req)
	assert.Equal(t, nil, err, "app.Test(req)")
	assert.Equal(t, 200, resp.StatusCode, "Status code")
	assert.Equal(t, "deflate", resp.Header.Get(routing.HeaderContentEncoding))

	// Validate that the file size has shrunk
	body, err := io.ReadAll(resp.Body)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, len(body) < len(filedata))
}

func Test_Compress_Brotli(t *testing.T) {
	t.Parallel()
	r := routing.New()

	r.Use(New())

	r.Get("/", func(c *routing.Ctx) error {
		c.Response.SetBodyRaw(filedata)
		return nil
	})

	req := httptest.NewRequest(routing.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "br")

	resp, err := r.Test(req, 10000)
	assert.Equal(t, nil, err, "app.Test(req)")
	assert.Equal(t, 200, resp.StatusCode, "Status code")
	assert.Equal(t, "br", resp.Header.Get(routing.HeaderContentEncoding))

	// Validate that the file size has shrunk
	body, err := io.ReadAll(resp.Body)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, len(body) < len(filedata))
}

func Test_Compress_Disabled(t *testing.T) {
	t.Parallel()
	r := routing.New()

	r.Use(New(Config{Level: LevelDisabled}))

	r.Get("/", func(c *routing.Ctx) error {
		c.Response.SetBodyRaw(filedata)
		return nil
	})

	req := httptest.NewRequest(routing.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "br")

	resp, err := r.Test(req)
	assert.Equal(t, nil, err, "app.Test(req)")
	assert.Equal(t, 200, resp.StatusCode, "Status code")
	assert.Equal(t, "", resp.Header.Get(routing.HeaderContentEncoding))

	// Validate the file size is not shrunk
	body, err := io.ReadAll(resp.Body)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, len(body) == len(filedata))
}

func Test_Compress_Next_Error(t *testing.T) {
	t.Parallel()
	r := routing.New()

	r.Use(New())

	r.Get("/", func(c *routing.Ctx) error {
		return errors.New("next error")
	})

	req := httptest.NewRequest(routing.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := r.Test(req)
	assert.Equal(t, nil, err, "app.Test(req)")
	assert.Equal(t, 500, resp.StatusCode, "Status code")
	assert.Equal(t, "", resp.Header.Get(routing.HeaderContentEncoding))

	body, err := io.ReadAll(resp.Body)
	assert.Equal(t, nil, err)
	assert.Equal(t, "next error", string(body))
}

// go test -run Test_Compress_Next
func Test_Compress_Next(t *testing.T) {
	t.Parallel()
	r := routing.New()
	r.Use(New(Config{
		Next: func(_ *routing.Ctx) bool {
			return true
		},
	}))

	resp, err := r.Test(httptest.NewRequest(routing.MethodGet, "/", nil))
	assert.Equal(t, nil, err)
	assert.Equal(t, routing.StatusNotFound, resp.StatusCode)
}
