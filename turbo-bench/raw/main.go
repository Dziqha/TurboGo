package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/valyala/fasthttp"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	handler := func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("text/plain")
		ctx.SetBodyString("Hello, World!")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	srv := &fasthttp.Server{
		Handler:                       handler,
		Name:                          "Raw",
		Logger:                        log.New(io.Discard, "", 0),
		LogAllErrors:                  false,
		ReadBufferSize:                4096,
		WriteBufferSize:               4096,
		Concurrency:                   256 * 1024,
		MaxConnsPerIP:                 10000,
		MaxRequestsPerConn:            0,
		DisableKeepalive:              false,
		TCPKeepalive:                  false,
		ReduceMemoryUsage:             false,
		NoDefaultServerHeader:         true,
		NoDefaultDate:                 true,
		NoDefaultContentType:          false,
		DisableHeaderNamesNormalizing: true,
		DisablePreParseMultipartForm:  true,
		StreamRequestBody:             false,
		GetOnly:                       false,
	}

	fmt.Printf("Raw fasthttp listening on port %s\n", port)
	if err := srv.ListenAndServe(":" + port); err != nil {
		panic(err)
	}
}
