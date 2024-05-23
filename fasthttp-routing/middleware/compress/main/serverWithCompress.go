package main

import (
	"io"
	"io/fs"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"sync"

	routing "fasthttp-routing"
	"fasthttp-routing/middleware/compress"
	"github.com/newacorn/fasthttp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"helpers/unsafefn"
)

type BigfilePool struct {
	fileName string
	pool     sync.Pool
}

func NewBigPool(name string) *BigfilePool {
	p := BigfilePool{fileName: name, pool: sync.Pool{
		New: func() any {
			b := bigFileReader{}
			f, err := os.Open(name)
			if err != nil {
				log.Fatal().Err(err).Send()
				b.err = err
				return b
			}
			b.f = f
			b.r = f
			return &b
		},
	}}
	return &p
}

var cur, _ = os.Getwd()

var p1 = NewBigPool("./README.md")

func main() {
	println(cur)

	println(runtime.GOMAXPROCS(16))
	go func() {
		log.Fatal().Err(http.ListenAndServe(":6666", nil)).Send()
	}()
	log.Fatal().Err(fasthttp.ListenAndServe(":7777", getRouter())).Send()
}

func getRouter() fasthttp.RequestHandler {
	f, err := os.Open("./README.md")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	r := routing.New()
	// size, err := f.Seek(0, io.SeekEnd)
	r.Use(compress.New())
	r.Get("/bench", func(ctx *routing.Ctx) error {
		_, err := ctx.Write(unsafefn.StoB("hello world"))
		return err
	})
	r.Get("/readme.md", func(ctx *routing.Ctx) error {
		/*
			f, err := os.Open("./README.md")
			if err != nil {
				log.Fatal().Stack().Err(err).Send()
			}
		*/
		b := p1.pool.Get().(*bigFileReader)
		ctx.SetBodyStream(b, int(size))
		return nil
		/*
			ctx.Response.SetBodyStreamWriter(func(w *bufio.Writer) {
				_, err = io.Copy(w, f)
				if err != nil {
					log.Fatal().Stack().Err(err)
				}
			})
		*/
	})
	return r.HandleRequest
}

type bigFileReader struct {
	f   fs.File
	r   io.Reader
	lr  io.LimitedReader
	err error
}

func (r *bigFileReader) UpdateByteRange(startPos, endPos int) error {
	seeker, ok := r.f.(io.Seeker)
	if !ok {
		return errors.New("must implement io.Seeker")
	}
	if _, err := seeker.Seek(int64(startPos), io.SeekStart); err != nil {
		return err
	}
	r.r = &r.lr
	r.lr.R = r.f
	r.lr.N = int64(endPos - startPos + 1)
	return nil
}

func (r *bigFileReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r *bigFileReader) WriteTo(w io.Writer) (int64, error) {
	if rf, ok := w.(io.ReaderFrom); ok {
		// fast path. Send file must be triggered
		return rf.ReadFrom(r.r)
	}

	// slow path
	return copyZeroAlloc(w, r.r)
}

func (r *bigFileReader) Close() error {
	r.r = r.f
	seeker, ok := r.f.(io.Seeker)
	if !ok {
		_ = r.f.Close()
		return errors.New("must implement io.Seeker")
	}
	n, err := seeker.Seek(0, io.SeekStart)
	if err == nil {
		if n != 0 {
			_ = r.f.Close()
			err = errors.New("bug: File.Seek(0, io.SeekStart) returned (non-zero, nil)")
		}
		p1.pool.Put(r)
	} else {
		_ = r.f.Close()
	}
	return err
}
func copyZeroAlloc(w io.Writer, r io.Reader) (int64, error) {
	if wt, ok := r.(io.WriterTo); ok {
		return wt.WriteTo(w)
	}
	if rt, ok := w.(io.ReaderFrom); ok {
		return rt.ReadFrom(r)
	}
	vbuf := fasthttp.CopyBufPool.Get()
	buf := vbuf.([]byte)
	n, err := io.CopyBuffer(w, r, buf)
	fasthttp.CopyBufPool.Put(vbuf)
	return n, err
}
