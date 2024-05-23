package compress

import (
	"bufio"
	"io"
	"net"
	"os"
	"testing"

	routing "fasthttp-routing"
	"github.com/newacorn/fasthttp"
	"github.com/rs/zerolog/log"
)

func BenchmarkBrio(b *testing.B) {
	r := getRouter()
	requestWithDifferentEncoding("br", b, r)
}

func requestWithDifferentEncoding(encoding string, b *testing.B, handler fasthttp.RequestHandler) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI("/readme.md")
	fctx := fasthttp.RequestCtx{}
	fctx.Init(req, nil, nil)
	fctx.SetConn(&fakeAddrer{})
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		handler(&fctx)
		fctx.Response.Reset()
	}
	// fasthttp.ReleaseRequest(req)
}

func getRouter() fasthttp.RequestHandler {
	f, err := os.Open("../../README.md")
	if err != nil {
		log.Fatal().Stack().Err(err)
	}
	r := routing.New()
	r.Use(New())
	r.Get("/readme.md", func(ctx *routing.Ctx) error {
		ctx.Response.SetBodyStreamWriter(func(w *bufio.Writer) {
			_, err = io.Copy(w, f)
			if err != nil {
				log.Fatal().Stack().Err(err)
			}
		})
		return nil
	})
	return r.HandleRequest
}

func (fa *fakeAddrer) RemoteAddr() net.Addr {
	return fa.raddr
}

func (fa *fakeAddrer) LocalAddr() net.Addr {
	return fa.laddr
}

func (fa *fakeAddrer) Read(p []byte) (int, error) {
	// developer sanity-check
	panic("BUG: unexpected Read call")
}

func (fa *fakeAddrer) Write(p []byte) (int, error) {
	// developer sanity-check
	panic("BUG: unexpected Write call")
}

func (fa *fakeAddrer) Close() error {
	// developer sanity-check
	panic("BUG: unexpected Close call")
}

type fakeAddrer struct {
	net.Conn
	laddr net.Addr
	raddr net.Addr
}

func TestWithWrk(t *testing.T) {
	log.Fatal().Err(fasthttp.ListenAndServe(":7777", getRouter()))
}
