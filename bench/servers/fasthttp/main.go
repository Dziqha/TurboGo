package main

import (
	"encoding/json"
	"net/http"
	_ "net/http/pprof"
	"os"

	"bench.local/db"

	"github.com/valyala/fasthttp"
)

type Message struct {
	Message string `json:"message"`
}

func main() {
	go db.Init()
	if os.Getenv("PPROF") == "1" {
		go http.ListenAndServe("127.0.0.1:9091", nil)
	}

	handler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		switch {
		case path == "/plaintext":
			ctx.SetContentType("text/plain")
			ctx.SetBodyString("Hello, World!")
		case path == "/json":
			ctx.SetContentType("application/json")
			res, _ := json.Marshal(Message{Message: "Hello, World!"})
			ctx.SetBody(res)
		case len(path) > 6 && path[:6] == "/user/":
			ctx.SetBodyString("user:" + path[6:])
		case path == "/db":
			if !db.Ready() {
				ctx.SetStatusCode(503)
				return
			}
			w, err := db.FetchOne(db.RandID())
			if err != nil {
				ctx.SetStatusCode(500)
				return
			}
			ctx.SetContentType("application/json")
			res, _ := json.Marshal(w)
			ctx.SetBody(res)
		case path == "/queries":
			if !db.Ready() {
				ctx.SetStatusCode(503)
				return
			}
			n := db.ParseQueryCount(string(ctx.QueryArgs().Peek("queries")))
			worlds, err := db.FetchMany(db.RandIDs(n))
			if err != nil {
				ctx.SetStatusCode(500)
				return
			}
			ctx.SetContentType("application/json")
			res, _ := json.Marshal(worlds)
			ctx.SetBody(res)
		case path == "/fortunes":
			if !db.Ready() {
				ctx.SetStatusCode(503)
				return
			}
			fortunes, err := db.AllFortunes()
			if err != nil {
				ctx.SetStatusCode(500)
				return
			}
			ctx.SetContentType("text/html; charset=utf-8")
			ctx.SetBodyString(db.RenderFortunes(fortunes))
		case path == "/update":
			if !db.Ready() {
				ctx.SetStatusCode(503)
				return
			}
			n := db.ParseQueryCount(string(ctx.QueryArgs().Peek("queries")))
			ids := db.RandIDs(n)
			worlds, err := db.FetchMany(ids)
			if err != nil {
				ctx.SetStatusCode(500)
				return
			}
			for i := range worlds {
				worlds[i].RandomNumber = db.RandID()
			}
			if err := db.UpdateAll(worlds); err != nil {
				ctx.SetStatusCode(500)
				return
			}
			ctx.SetContentType("application/json")
			res, _ := json.Marshal(worlds)
			ctx.SetBody(res)
		default:
			ctx.SetStatusCode(404)
			ctx.SetBodyString("not found")
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &fasthttp.Server{
		Handler:                       handler,
		TCPKeepalive:                  false,
		NoDefaultServerHeader:         true,
		NoDefaultDate:                 true,
		NoDefaultContentType:          true,
		DisableHeaderNamesNormalizing: true,
	}
	if err := srv.ListenAndServe(":" + port); err != nil {
		panic(err)
	}
}