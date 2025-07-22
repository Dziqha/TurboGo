// package main

// import (
// 	"fmt"
// 	"runtime"
// 	"sync"
// 	"sync/atomic"
// 	"time"
// 	"github.com/Dziqha/TurboGo/core"
// 	"github.com/Dziqha/TurboGo/internal/router"
// )

// type Context = core.Context
// // Simulasi router kamu
// var ur = router.NewUltimateRouter()

// func main() {
// 	staticRoutes := []string{
// 		"/", "/about", "/contact", "/api/health", "/api/status", "/api/version",
// 		"/users", "/products", "/orders", "/payments", "/dashboard", "/profile",
// 		"/settings", "/admin", "/login", "/logout", "/register", "/forgot-password",
// 		"/api/users", "/api/products", "/api/orders", "/api/payments", "/api/auth",
// 		"/api/admin", "/api/config", "/api/logs", "/api/metrics", "/api/backup",
// 		"/docs", "/help", "/support", "/feedback", "/pricing", "/features",
// 		"/blog", "/news", "/events", "/careers", "/team", "/investors",
// 		"/legal", "/privacy", "/terms", "/cookies", "/security", "/status",
// 	}

// 	for _, route := range staticRoutes {
// 		ur.AddStatic("GET", route, dummyHandler, &core.Route{Path: route, Method: "GET"})
// 	}

// 	paramRoutes := []string{
// 		"/user/:id", "/product/:id", "/order/:id", "/api/user/:id", "/api/product/:id",
// 		"/api/user/:id/posts", "/api/user/:id/orders", "/api/product/:id/reviews",
// 		"/api/order/:id/items", "/api/payment/:id/status", "/api/admin/:id/logs",
// 		"/blog/:slug", "/docs/:category/:page", "/help/:topic", "/support/:ticket",
// 	}

// 	for _, route := range paramRoutes {
// 		ur.AddParametric("GET", route, dummyHandler, &core.Route{Path: route, Method: "GET"})
// 	}

// 	wildcardPrefixes := []string{
// 		"/static/", "/uploads/", "/assets/", "/images/", "/files/", "/downloads/",
// 		"/cdn/", "/cache/", "/tmp/", "/public/",
// 	}

// 	for _, prefix := range wildcardPrefixes {
// 		ur.AddWildcard("GET", prefix, dummyHandler, &core.Route{Path: prefix + "*", Method: "GET"})
// 	}

// 	testPaths := []string{
// 		"/", "/api/health", "/users", "/products", "/dashboard", "/api/auth",
// 		"/settings", "/admin", "/login", "/api/users", "/api/products", "/docs",
// 		"/user/123", "/product/456", "/order/789", "/api/user/111", "/blog/my-post",
// 		"/api/user/333/posts", "/docs/api/getting-started", "/help/billing",
// 		"/static/css/app.css", "/static/js/bundle.js", "/uploads/avatar.jpg",
// 		"/assets/images/logo.png", "/cdn/libs/jquery.min.js", "/cache/page.html",
// 		"/nonexistent", "/api/nonexistent", "/random/path", "/does/not/exist",
// 	}

// 	fmt.Println("=== ULTIMATE WARM UP ===")
// 	for i := 0; i < 1_000_000; i++ {
// 		for _, path := range testPaths[:20] {
// 			ur.Find("GET", path)
// 		}
// 	}

// 	fmt.Println("\n=== SINGLE-THREADED BENCHMARK ===")
// 	for _, path := range testPaths[:12] {
// 		start := time.Now()
// 		for i := 0; i < 2_000_000; i++ {
// 			ur.Find("GET", path)
// 		}
// 		dur := time.Since(start)
// 		fmt.Printf("Path %-30s: %8s (%.2f ns/op)\n", path, dur, float64(dur.Nanoseconds())/2_000_000)
// 	}

// 	ultimateBenchmark(ur, testPaths, 10, 200_000)
// 	ultimateBenchmark(ur, testPaths, 100, 20_000)
// 	ultimateBenchmark(ur, testPaths, 1000, 2_000)
// 	ultimateBenchmark(ur, testPaths, 5000, 400)

// 	ultimateStressTest(ur, testPaths, 10*time.Second)

// 	fmt.Println("\n=== ULTIMATE STATISTICS ===")
// 	stats := ur.GetAdvancedStats()

// 	for key, value := range stats {
// 		switch v := value.(type) {
// 		case float64:
// 			fmt.Printf("%s: %.2f\n", key, v)
// 		case int64:
// 			fmt.Printf("%s: %d\n", key, v)
// 		}
// 	}

// 	var m runtime.MemStats
// 	runtime.ReadMemStats(&m)
// 	fmt.Printf("\n=== MEMORY ANALYSIS ===\n")
// 	fmt.Printf("Current memory usage: %.2f MB\n", float64(m.Alloc)/1024/1024)
// 	fmt.Printf("Peak memory usage: %.2f MB\n", float64(m.Sys)/1024/1024)
// 	fmt.Printf("Total allocations: %.2f GB\n", float64(m.TotalAlloc)/1024/1024/1024)
// 	fmt.Printf("GC runs: %d\n", m.NumGC)
// 	if m.NumGC > 0 {
// 		fmt.Printf("Average GC pause: %.2f ms\n", float64(m.PauseTotalNs)/float64(m.NumGC)/1e6)
// 	}
// 	fmt.Println("\n🔥🚀 ULTIMATE BENCHMARK COMPLETED! 🚀🔥")
// }

// func dummyHandler(c *core.Context) {
// 	c.Text(200, "OK")
// }

// func ultimateBenchmark(r *router.UltimateRouter, paths []string, goroutines, perGoroutine int) {
// 	fmt.Printf("\n=== CONCURRENT %d×%d Benchmark ===\n", goroutines, perGoroutine)
// 	var wg sync.WaitGroup
// 	start := time.Now()
// 	for i := 0; i < goroutines; i++ {
// 		wg.Add(1)
// 		go func(id int) {
// 			defer wg.Done()
// 			for j := 0; j < perGoroutine; j++ {
// 				path := paths[(id+j)%len(paths)]
// 				r.Find("GET", path)
// 			}
// 		}(i)
// 	}
// 	wg.Wait()
// 	dur := time.Since(start)
// 	fmt.Printf("Took %v | %.2f ns/op\n", dur, float64(dur.Nanoseconds())/float64(goroutines*perGoroutine))
// }

// func ultimateStressTest(r *router.UltimateRouter, paths []string, dur time.Duration) {
// 	fmt.Println("\n=== ULTIMATE STRESS TEST ===")
// 	stop := time.Now().Add(dur)
// 	var count int64
// 	var wg sync.WaitGroup
// 	cpus := runtime.NumCPU()

// 	for i := 0; i < cpus; i++ {
// 		wg.Add(1)
// 		go func(id int) {
// 			defer wg.Done()
// 			for time.Now().Before(stop) {
// 				for _, path := range paths {
// 					r.Find("GET", path)
// 					atomic.AddInt64(&count, 1)
// 				}
// 			}
// 		}(i)
// 	}

// 	wg.Wait()
// 	fmt.Printf("Total ops: %d in %v => %.2f req/sec\n", count, dur, float64(count)/dur.Seconds())
// }

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
