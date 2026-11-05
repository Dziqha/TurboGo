package main

import (
	"os"

	"bench.local/db"

	"github.com/gin-gonic/gin"
)

type Message struct {
	Message string `json:"message"`
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	db.Init()

	app := gin.New()
	app.Use(gin.Recovery())

	app.GET("/plaintext", func(c *gin.Context) {
		c.String(200, "Hello, World!")
	})

	app.GET("/json", func(c *gin.Context) {
		c.JSON(200, Message{Message: "Hello, World!"})
	})

	app.GET("/user/:id", func(c *gin.Context) {
		c.String(200, "user:"+c.Param("id"))
	})

	app.GET("/db", func(c *gin.Context) {
		if !db.Ready() {
			c.String(503, "")
			return
		}
		w, err := db.FetchOne(db.RandID())
		if err != nil {
			c.String(500, "")
			return
		}
		c.JSON(200, w)
	})

	app.GET("/queries", func(c *gin.Context) {
		if !db.Ready() {
			c.String(503, "")
			return
		}
		n := db.ParseQueryCount(c.Query("queries"))
		worlds, err := db.FetchMany(db.RandIDs(n))
		if err != nil {
			c.String(500, "")
			return
		}
		c.JSON(200, worlds)
	})

	app.GET("/fortunes", func(c *gin.Context) {
		if !db.Ready() {
			c.String(503, "")
			return
		}
		fortunes, err := db.AllFortunes()
		if err != nil {
			c.String(500, "")
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, db.RenderFortunes(fortunes))
	})

	app.GET("/update", func(c *gin.Context) {
		if !db.Ready() {
			c.String(503, "")
			return
		}
		n := db.ParseQueryCount(c.Query("queries"))
		ids := db.RandIDs(n)
		worlds, err := db.FetchMany(ids)
		if err != nil {
			c.String(500, "")
			return
		}
		for i := range worlds {
			worlds[i].RandomNumber = db.RandID()
		}
		if err := db.UpdateAll(worlds); err != nil {
			c.String(500, "")
			return
		}
		c.JSON(200, worlds)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := app.Run(":" + port); err != nil {
		panic(err)
	}
}