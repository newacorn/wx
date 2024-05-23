package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"

	routing "fasthttp-routing"
	"fasthttp-routing/middleware/csrf"
	"fasthttp-routing/middleware/session"
	"github.com/newacorn/fasthttp"
	"github.com/rs/zerolog/log"
)

func main88() {
	r := routing.New()
	cfg := session.DefCfg
	cfg.Codec = session.JSONCodec{}
	go func() {
		log.Fatal().Err(http.ListenAndServe(":6666", nil))
	}()
	r.Use(session.New(&cfg))
	r.Use(csrf.New())
	r.Get("/cookie", func(ctx *routing.Ctx) error {
		ctx.Response.Header.Set("Cache-Control", "max-age=604800")
		ctx.SetBodyString("hello world i am cached.")
		return nil
	})
	r.Get("/hello", func(ctx *routing.Ctx) (err error) {
		data := ctx.UserValue(session.ContextKey).(*session.Data)
		_ = data
		data.Put("foo", "bar")
		ctx.SetBodyString("Hello World!")
		return
	})

	r.Post("/h", func(ctx *routing.Ctx) error {
		return nil
	})
	cert, err := tls.LoadX509KeyPair("/Users/acorn/workspace/certs/newacorn.crt", "/Users/acorn/workspace/certs/newacorn.key")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	keyLogWriter, err := os.Create("/media/psf/Home/Desktop/keylog.log")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	defer func() { _ = keyLogWriter.Close() }()
	server := fasthttp.Server{Handler: r.HandleRequest /*TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}, KeyLogWriter: keyLogWriter}*/}

	tcpListenConfig := net.ListenConfig{KeepAlive: -1}
	tcpListener, err := tcpListenConfig.Listen(context.Background(), "tcp", ":7777")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	tlsListener := tls.NewListener(tcpListener, &tls.Config{Certificates: []tls.Certificate{cert}, KeyLogWriter: keyLogWriter})
	err = server.Serve(tlsListener)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	// err = server.ListenAndServe(":7777")
	/*
		err = server.ListenAndServeTLS(":7777", "", "")
		if err != nil {
			log.Fatal().Err(err)
		}
	*/
	// log.Fatal().Err(fasthttp.ListenAndServe(":7777", r.HandleRequest)).Send()
	// fasthttp.
}
