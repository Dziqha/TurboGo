package main

import (
	"github.com/Dziqha/TurboGo"
	"github.com/Dziqha/TurboGo/core"
	"github.com/Dziqha/TurboGo/middleware"
	_ "fmt"
	"time"
)

func main() {
	app := TurboGo.New()
	app.Use(
		middleware.Recover(), // selalu paling atas
		middleware.Logger(),  // log seluruh proses
		middleware.Auth(),    // auth sebelum cache
		
	)
	
	
	// Route demo
	app.Get("/hello", func(c *core.Context) {

		c.Text(200, "Hello TurboGo!")
	}).NoCache()
	
	app.Get("/test-redis", func(c *core.Context) {
		// Set key (tanpa err karena tidak return apa-apa)
		c.Redis.Memory.Set("test", []byte("TurboGo"), 10*time.Second)
	
		// Get key (misal nil = not found)
		val, _ := c.Redis.Memory.Get("test")
		if val == nil {
			c.JSON(404, map[string]string{"error": "not found"})
			return
		}
	
		// Return result
		c.JSON(200, map[string]string{
			"redis_value": string(val),
		})
	})

	app.Get("/ttl/:key", func(c *core.Context) {
		key := c.Param("key")
		ttl := c.Redis.Memory.TTL(key)
	
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
		c.Redis.Memory.Range(func(k string, _ []byte) {
			keys = append(keys, k)
		})
		c.JSON(200, keys)
	})
	

	// Start server
	app.Listen(":8080")
}
