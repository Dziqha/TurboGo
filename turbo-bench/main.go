package main

import (
	"fmt"
	"os"

	"github.com/Dziqha/TurboGo"
	"github.com/Dziqha/TurboGo/core"
)

func main() {
	core.DisableLogger = true

	app := TurboGo.New()

	app.Get("/plaintext", PlaintextHandler)
	app.Get("/json", JSONHandler)
	app.Get("/db", DBHandler)
	app.Get("/queries", QueriesHandler)
	app.Get("/fortunes", FortunesHandler)
	app.Get("/update", UpdateHandler)

	go initDB()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("TurboGo benchmark server listening on port %s\n", port)
	if err := app.RunServer(":" + port); err != nil {
		panic(err)
	}
}
