package main

import (
	"os"

	"bench.local/db"

	"github.com/gofiber/fiber/v2"
)

type Message struct {
	Message string `json:"message"`
}

func main() {
	go db.Init()

	app := fiber.New(fiber.Config{
		DisableDefaultDate:       true,
		DisableHeaderNormalizing: true,
		DisableStartupMessage:    true,
		ServerHeader:             "",
	})

	app.Get("/plaintext", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Get("/json", func(c *fiber.Ctx) error {
		return c.JSON(Message{Message: "Hello, World!"})
	})

	app.Get("/user/:id", func(c *fiber.Ctx) error {
		return c.SendString("user:" + c.Params("id"))
	})

	app.Get("/db", func(c *fiber.Ctx) error {
		if !db.Ready() {
			return c.SendStatus(503)
		}
		w, err := db.FetchOne(db.RandID())
		if err != nil {
			return c.SendStatus(500)
		}
		return c.JSON(w)
	})

	app.Get("/queries", func(c *fiber.Ctx) error {
		if !db.Ready() {
			return c.SendStatus(503)
		}
		n := db.ParseQueryCount(c.Query("queries"))
		worlds, err := db.FetchMany(db.RandIDs(n))
		if err != nil {
			return c.SendStatus(500)
		}
		return c.JSON(worlds)
	})

	app.Get("/fortunes", func(c *fiber.Ctx) error {
		if !db.Ready() {
			return c.SendStatus(503)
		}
		fortunes, err := db.AllFortunes()
		if err != nil {
			return c.SendStatus(500)
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(db.RenderFortunes(fortunes))
	})

	app.Get("/update", func(c *fiber.Ctx) error {
		if !db.Ready() {
			return c.SendStatus(503)
		}
		n := db.ParseQueryCount(c.Query("queries"))
		ids := db.RandIDs(n)
		worlds, err := db.FetchMany(ids)
		if err != nil {
			return c.SendStatus(500)
		}
		for i := range worlds {
			worlds[i].RandomNumber = db.RandID()
		}
		if err := db.UpdateAll(worlds); err != nil {
			return c.SendStatus(500)
		}
		return c.JSON(worlds)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := app.Listen(":" + port); err != nil {
		panic(err)
	}
}