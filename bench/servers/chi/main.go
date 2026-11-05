package main

import (
	"encoding/json"
	"net/http"
	"os"

	"bench.local/db"

	"github.com/go-chi/chi/v5"
)

type Message struct {
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func main() {
	db.Init()
	r := chi.NewRouter()

	r.Get("/plaintext", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Hello, World!"))
	})

	r.Get("/json", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, Message{Message: "Hello, World!"})
	})

	r.Get("/user/{id}", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("user:" + chi.URLParam(req, "id")))
	})

	r.Get("/db", func(w http.ResponseWriter, _ *http.Request) {
		if !db.Ready() {
			w.WriteHeader(503)
			return
		}
		world, err := db.FetchOne(db.RandID())
		if err != nil {
			w.WriteHeader(500)
			return
		}
		writeJSON(w, 200, world)
	})

	r.Get("/queries", func(w http.ResponseWriter, req *http.Request) {
		if !db.Ready() {
			w.WriteHeader(503)
			return
		}
		n := db.ParseQueryCount(req.URL.Query().Get("queries"))
		worlds, err := db.FetchMany(db.RandIDs(n))
		if err != nil {
			w.WriteHeader(500)
			return
		}
		writeJSON(w, 200, worlds)
	})

	r.Get("/fortunes", func(w http.ResponseWriter, _ *http.Request) {
		if !db.Ready() {
			w.WriteHeader(503)
			return
		}
		fortunes, err := db.AllFortunes()
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(db.RenderFortunes(fortunes)))
	})

	r.Get("/update", func(w http.ResponseWriter, req *http.Request) {
		if !db.Ready() {
			w.WriteHeader(503)
			return
		}
		n := db.ParseQueryCount(req.URL.Query().Get("queries"))
		ids := db.RandIDs(n)
		worlds, err := db.FetchMany(ids)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		for i := range worlds {
			worlds[i].RandomNumber = db.RandID()
		}
		if err := db.UpdateAll(worlds); err != nil {
			w.WriteHeader(500)
			return
		}
		writeJSON(w, 200, worlds)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := http.ListenAndServe(":"+port, r); err != nil {
		panic(err)
	}
}