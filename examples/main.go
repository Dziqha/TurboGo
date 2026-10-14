package main

import (
	"fmt"
	"time"

	"github.com/Dziqha/TurboGo"
	"github.com/Dziqha/TurboGo/core"
	"github.com/Dziqha/TurboGo/examples/controller"
)

func main() {

	app := TurboGo.New().WithCache().WithQueue().WithPubsub()
	core.DisableLogger = false
	controller.Quehandler(app.EngineCtx)
	go controller.PubsubHandler(app.EngineCtx)
	app.Post("/hai", controller.CreateUserHandler)
	app.Get("/", func(ctx *core.Context) {
		key := "my-key"
		val, found := ctx.Cache.Memory.Get(key)
		if found {
			fmt.Println("✅ Cache HIT:", key)
			ctx.Text(200, "From Cache: "+string(val))
			return
		}

		fmt.Println("🔄 Cache MISS:", key)
		result := "Hello, TurboGo!"
		ctx.Cache.Memory.Set(key, []byte(result), 10*time.Second)
		ctx.Text(200, "Generated: "+result)
	})

	app.RunServer(":8080")

}
