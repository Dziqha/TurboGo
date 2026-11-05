package main

import (
	"net/http"
	"os"

	"bench.local/db"

	"github.com/labstack/echo/v5"
)

type Message struct {
	Message string `json:"message"`
}

func main() {
	db.Init()
	app := echo.New()

	app.GET("/plaintext", func(c *echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	app.GET("/json", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, Message{Message: "Hello, World!"})
	})

	app.GET("/user/:id", func(c *echo.Context) error {
		return c.String(http.StatusOK, "user:"+c.Param("id"))
	})

	app.GET("/db", func(c *echo.Context) error {
		if !db.Ready() {
			return c.String(http.StatusServiceUnavailable, "")
		}
		w, err := db.FetchOne(db.RandID())
		if err != nil {
			return c.String(http.StatusInternalServerError, "")
		}
		return c.JSON(http.StatusOK, w)
	})

	app.GET("/queries", func(c *echo.Context) error {
		if !db.Ready() {
			return c.String(http.StatusServiceUnavailable, "")
		}
		n := db.ParseQueryCount(c.QueryParam("queries"))
		worlds, err := db.FetchMany(db.RandIDs(n))
		if err != nil {
			return c.String(http.StatusInternalServerError, "")
		}
		return c.JSON(http.StatusOK, worlds)
	})

	app.GET("/fortunes", func(c *echo.Context) error {
		if !db.Ready() {
			return c.String(http.StatusServiceUnavailable, "")
		}
		fortunes, err := db.AllFortunes()
		if err != nil {
			return c.String(http.StatusInternalServerError, "")
		}
		c.Response().Header().Set(echo.HeaderContentType, "text/html; charset=utf-8")
		return c.String(http.StatusOK, db.RenderFortunes(fortunes))
	})

	app.GET("/update", func(c *echo.Context) error {
		if !db.Ready() {
			return c.String(http.StatusServiceUnavailable, "")
		}
		n := db.ParseQueryCount(c.QueryParam("queries"))
		ids := db.RandIDs(n)
		worlds, err := db.FetchMany(ids)
		if err != nil {
			return c.String(http.StatusInternalServerError, "")
		}
		for i := range worlds {
			worlds[i].RandomNumber = db.RandID()
		}
		if err := db.UpdateAll(worlds); err != nil {
			return c.String(http.StatusInternalServerError, "")
		}
		return c.JSON(http.StatusOK, worlds)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := app.Start(":" + port); err != nil {
		panic(err)
	}
}