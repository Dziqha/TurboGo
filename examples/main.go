package main

import (
	"fmt"
	"time"

	"github.com/Dziqha/TurboGo"
	"github.com/Dziqha/TurboGo/core"
	"github.com/Dziqha/TurboGo/middleware"
)

func PublicHandler(c *core.Context) {
	fmt.Println("🔥 PublicHandler jalan")
	c.JSON(200, map[string]any{
		"message": "this is public content",
	})
}

func PrivateHandler(c *core.Context) {
	c.JSON(200, map[string]any{
		"user": "admin",
	})
}

func HeavyHandler(c *core.Context) {
	// Simulasi proses berat
	time.Sleep(2 * time.Second)
	c.JSON(200, map[string]any{
		"data": "processed",
	})
}


func main() {
	app := TurboGo.New()
	secret := "supersecurekey123"
	app.Use(
		middleware.Recover(), 
		middleware.Auth(secret),   
		
	)	

	
	app.Get("/public", PublicHandler)

	// ⛔ No cache: override dengan .NoCache()
	app.Get("/private", PrivateHandler).NoCache()

	// ⚠️ POST normally no-cache, tapi bisa di-cache pakai .Cache()
	app.Post("/heavy", HeavyHandler).Cache(5 * time.Second)
	
	app.Get("/hello", func(c *core.Context) {

		c.Text(200, "Hello TurboGo!")
	}).NoCache()
	
	app.Get("/test-redis", func(c *core.Context) {
		c.Cache.Memory.Set("test", []byte("TurboGo"), 10*time.Second)
	
		val, _ := c.Cache.Memory.Get("test")
		if val == nil {
			c.JSON(404, map[string]string{"error": "not found"})
			return
		}
	
		c.JSON(200, map[string]string{
			"cache_value": string(val),
		})
	})

	app.Get("/ttl/:key", func(c *core.Context) {
		key := c.Param("key")
		ttl := c.Cache.Memory.TTL(key)
	
		switch {
		case ttl == -2*time.Second:
			c.JSON(404, map[string]string{"error": "key not found or expired"})
		case ttl == -1*time.Second:
			c.JSON(200, map[string]string{"ttl": "infinite"})
		default:
			c.JSON(200, map[string]string{"ttl": ttl.String()})
		}
	})
	
	
	app.Get("/debug/routes", func(c *core.Context) {
		routes := app.RoutesInfo()
		c.JSON(200, routes)
	}).Named("routes")
	
	app.Get("/debug/redis/keys", func(c *core.Context) {
		keys := make([]string, 0)
		c.Cache.Memory.Range(func(k string, _ []byte) {
			keys = append(keys, k)
		})
		c.JSON(200, keys)
	})
	

	app.Listen(":8080")
}
