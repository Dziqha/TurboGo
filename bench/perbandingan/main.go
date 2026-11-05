package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Dziqha/TurboGo"
	"github.com/Dziqha/TurboGo/core"
	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
)

// 🔥 OPTIMIZED CONFIGURATION
var (
	testDuration      = 10 * time.Second
	warmupDuration    = 2 * time.Second
	concurrencyLevels = []int{1, 5, 10, 25, 50} // Progressive load
	basePort          = 8080
)

// Simplified test paths
var testPaths = []string{
	"/",
	"/api/health",
	"/api/v1/users",
	"/user/123",
	"/api/test",
	"/dashboard",
	"/search",
}

type BenchmarkResult struct {
	Framework        string                     `json:"framework"`
	TotalRequests    int64                      `json:"total_requests"`
	SuccessRequests  int64                      `json:"success_requests"`
	FailedRequests   int64                      `json:"failed_requests"`
	RequestsPerSec   float64                    `json:"requests_per_second"`
	AvgLatency       time.Duration              `json:"avg_latency_ns"`
	P95Latency       time.Duration              `json:"p95_latency_ns"`
	MemoryUsage      int64                      `json:"memory_usage_bytes"`
	GoroutineCount   int                        `json:"goroutine_count"`
	ConcurrencyTests map[int]*ConcurrencyResult `json:"concurrency_tests"`
}

type ConcurrencyResult struct {
	Concurrency    int           `json:"concurrency"`
	RequestsPerSec float64       `json:"requests_per_second"`
	AvgLatency     time.Duration `json:"avg_latency_ns"`
	P95Latency     time.Duration `json:"p95_latency_ns"`
	ErrorRate      float64       `json:"error_rate"`
	SuccessRate    float64       `json:"success_rate"`
}

type LatencyStats struct {
	latencies []time.Duration
	mu        sync.Mutex
}

func (ls *LatencyStats) Add(latency time.Duration) {
	ls.mu.Lock()
	if len(ls.latencies) < 10000 { // Increased capacity
		ls.latencies = append(ls.latencies, latency)
	}
	ls.mu.Unlock()
}

func (ls *LatencyStats) Calculate() (min, max, avg, p50, p90, p95 time.Duration) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if len(ls.latencies) == 0 {
		return 0, 0, 0, 0, 0, 0
	}

	sort.Slice(ls.latencies, func(i, j int) bool {
		return ls.latencies[i] < ls.latencies[j]
	})

	min = ls.latencies[0]
	max = ls.latencies[len(ls.latencies)-1]

	var total time.Duration
	for _, lat := range ls.latencies {
		total += lat
	}
	avg = total / time.Duration(len(ls.latencies))

	p50 = ls.latencies[int(float64(len(ls.latencies))*0.5)]
	p90 = ls.latencies[int(float64(len(ls.latencies))*0.9)]
	p95 = ls.latencies[int(float64(len(ls.latencies))*0.95)]

	return
}

// 🚀 IMPROVED HTTP CLIENT - NO RATE LIMITING
func createOptimizedClient(concurrency int) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second, // Increased timeout
		Transport: &http.Transport{
			MaxIdleConns:        200, // Higher limits
			MaxIdleConnsPerHost: concurrency * 2,
			MaxConnsPerHost:     concurrency * 3,
			IdleConnTimeout:     60 * time.Second,
			DisableKeepAlives:   false,
			DisableCompression:  true,
			// Remove timeouts that might cause issues
			TLSHandshakeTimeout:   0,
			ResponseHeaderTimeout: 0,
		},
	}
}

// 🔥 FIXED BENCHMARK FUNCTION - NO ARTIFICIAL RATE LIMITING
func benchmarkFramework(framework string, port int, concurrency int, duration time.Duration) *ConcurrencyResult {
	var (
		totalRequests   int64
		successRequests int64
		failedRequests  int64
		latencyStats    = &LatencyStats{}
	)

	client := createOptimizedClient(concurrency)
	defer func() {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup
	pathIndex := int64(0)

	// NO RATE LIMITER - Let it run free!
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Small stagger to prevent thundering herd
			time.Sleep(time.Duration(workerID) * 5 * time.Millisecond)

			localClient := &http.Client{
				Timeout: 10 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        10,
					MaxIdleConnsPerHost: 2,
					DisableKeepAlives:   false,
					DisableCompression:  true,
				},
			}
			defer func() {
				if transport, ok := localClient.Transport.(*http.Transport); ok {
					transport.CloseIdleConnections()
				}
			}()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// No rate limiting here - just go!
					pathIdx := atomic.AddInt64(&pathIndex, 1) % int64(len(testPaths))
					path := testPaths[pathIdx]

					url := fmt.Sprintf("http://localhost:%d%s", port, path)

					start := time.Now()
					resp, err := localClient.Get(url)
					latency := time.Since(start)

					atomic.AddInt64(&totalRequests, 1)
					latencyStats.Add(latency)

					if err != nil {
						atomic.AddInt64(&failedRequests, 1)
						// Small delay on error to prevent spam
						time.Sleep(1 * time.Millisecond)
						continue
					}

					if resp.StatusCode == http.StatusOK {
						atomic.AddInt64(&successRequests, 1)
					} else {
						atomic.AddInt64(&failedRequests, 1)
					}

					if resp.Body != nil {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}

					// Tiny delay to prevent CPU overload
					if concurrency > 10 {
						time.Sleep(100 * time.Microsecond)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	_, _, avg, _, _, p95 := latencyStats.Calculate()

	total := atomic.LoadInt64(&totalRequests)
	success := atomic.LoadInt64(&successRequests)
	failed := atomic.LoadInt64(&failedRequests)

	var errorRate float64
	if total > 0 {
		errorRate = float64(failed) / float64(total) * 100
	}

	requestsPerSec := float64(total) / duration.Seconds()

	return &ConcurrencyResult{
		Concurrency:    concurrency,
		RequestsPerSec: requestsPerSec,
		AvgLatency:     avg,
		P95Latency:     p95,
		ErrorRate:      errorRate,
		SuccessRate:    float64(success) / float64(total) * 100,
	}
}

// 🏠 SERVER SETUPS WITH BETTER ERROR HANDLING
func setupRobustFiber(port int) *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Prefork:               false,
		ServerHeader:          "",
		StrictRouting:         false,
		CaseSensitive:         false,
		Immutable:             false,
		UnescapePath:          false,
		ETag:                  false,
		BodyLimit:             4 * 1024 * 1024,
		Concurrency:           256 * 1024,
		ReadTimeout:           0,
		WriteTimeout:          0,
		IdleTimeout:           0,
		ReadBufferSize:        4096,
		WriteBufferSize:       4096,
		CompressedFileSuffix:  ".gz",
		ProxyHeader:           "",
		GETOnly:               false,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			return ctx.Status(500).SendString("Internal Server Error")
		},
	})

	// Simple routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Hello from Fiber!", "path": c.Path()})
	})
	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "framework": "fiber"})
	})
	app.Get("/api/v1/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"users": []string{"user1", "user2"}})
	})
	app.Get("/user/123", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"user_id": "123", "framework": "fiber"})
	})
	app.Get("/dashboard", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"page": "dashboard", "framework": "fiber"})
	})
	app.Get("/search", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"results": []string{}, "framework": "fiber"})
	})
	app.Post("/api/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"received": "ok"})
	})

	go func() {
		if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
			fmt.Printf("Fiber server error: %v\n", err)
		}
	}()
	time.Sleep(300 * time.Millisecond)
	return app
}

func setupRobustEcho(port int) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.GET("/", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{"message": "Hello from Echo!", "path": c.Request().URL.Path})
	})
	e.GET("/api/health", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{"status": "ok", "framework": "echo"})
	})
	e.GET("/api/v1/users", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{"users": []string{"user1", "user2"}})
	})
	e.GET("/user/123", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{"user_id": "123", "framework": "echo"})
	})
	e.GET("/dashboard", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{"page": "dashboard", "framework": "echo"})
	})
	e.GET("/search", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{"results": []string{}, "framework": "echo"})
	})
	e.POST("/api/test", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{"received": "ok"})
	})

	go func() {
		if err := e.Start(fmt.Sprintf(":%d", port)); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Echo server error: %v\n", err)
		}
	}()
	time.Sleep(300 * time.Millisecond)
	return e
}

func setupRobustGin(port int) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello from Gin!", "path": c.Request.URL.Path})
	})
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "framework": "gin"})
	})
	r.GET("/api/v1/users", func(c *gin.Context) {
		c.JSON(200, gin.H{"users": []string{"user1", "user2"}})
	})
	r.GET("/user/123", func(c *gin.Context) {
		c.JSON(200, gin.H{"user_id": "123", "framework": "gin"})
	})
	r.GET("/dashboard", func(c *gin.Context) {
		c.JSON(200, gin.H{"page": "dashboard", "framework": "gin"})
	})
	r.GET("/search", func(c *gin.Context) {
		c.JSON(200, gin.H{"results": []string{}, "framework": "gin"})
	})
	r.POST("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"received": "ok"})
	})

	go func() {
		if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
			fmt.Printf("Gin server error: %v\n", err)
		}
	}()
	time.Sleep(300 * time.Millisecond)
	return r
}

func setupRobustTurboGo(port int) {
	app := TurboGo.New()
	core.DisableLogger = true

	app.Get("/", func(ctx *core.Context) {
		ctx.JSON(200, map[string]interface{}{"message": "Hello from TurboGo!", "path": string(ctx.Ctx.Request.URI().Path())})
	})
	app.Get("/api/health", func(ctx *core.Context) {
		ctx.JSON(200, map[string]interface{}{"status": "ok", "framework": "turbogo"})
	})
	app.Get("/api/v1/users", func(ctx *core.Context) {
		ctx.JSON(200, map[string]interface{}{"users": []string{"user1", "user2"}})
	})
	app.Get("/user/123", func(ctx *core.Context) {
		ctx.JSON(200, map[string]interface{}{"user_id": "123", "framework": "turbogo"})
	})
	app.Get("/dashboard", func(ctx *core.Context) {
		ctx.JSON(200, map[string]interface{}{"page": "dashboard", "framework": "turbogo"})
	})
	app.Get("/search", func(ctx *core.Context) {
		ctx.JSON(200, map[string]interface{}{"results": []string{}, "framework": "turbogo"})
	})
	app.Post("/api/test", func(ctx *core.Context) {
		ctx.JSON(200, map[string]interface{}{"received": "ok"})
	})

	go func() {
		if err := app.RunServer(fmt.Sprintf(":%d", port)); err != nil {
			fmt.Printf("TurboGo server error: %v\n", err)
		}
	}()
	time.Sleep(300 * time.Millisecond)
}

func getMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	return int64(m.Alloc)
}

// 📊 ROBUST BENCHMARK RUNNER
func runRobustBenchmark(framework string, port int) *BenchmarkResult {
	fmt.Printf("🔥 Benchmarking %s...\n", framework)

	// Health check with retries
	client := createOptimizedClient(1)
	testURL := fmt.Sprintf("http://localhost:%d/", port)

	fmt.Printf("  🏥 Health check...\n")
	var healthOK bool
	for i := 0; i < 10; i++ {
		resp, err := client.Get(testURL)
		if err == nil && resp != nil && resp.StatusCode == 200 {
			if resp.Body != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
			healthOK = true
			break
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		time.Sleep(200 * time.Millisecond)
	}

	if !healthOK {
		fmt.Printf("  ❌ Health check failed for %s\n", framework)
		return &BenchmarkResult{
			Framework:        framework,
			ConcurrencyTests: make(map[int]*ConcurrencyResult),
		}
	}

	fmt.Printf("  ⏳ Warmup...\n")
	benchmarkFramework(framework, port, 2, warmupDuration)
	time.Sleep(500 * time.Millisecond)

	memBefore := getMemoryUsage()
	goroutinesBefore := runtime.NumGoroutine()

	result := &BenchmarkResult{
		Framework:        framework,
		ConcurrencyTests: make(map[int]*ConcurrencyResult),
	}

	var totalRPS float64
	var totalLatency time.Duration
	validTests := 0

	for _, concurrency := range concurrencyLevels {
		fmt.Printf("  📊 Testing concurrency: %d\n", concurrency)

		if concurrency > 10 {
			fmt.Printf("    💤 Brief pause...\n")
			time.Sleep(1 * time.Second)
		}

		concResult := benchmarkFramework(framework, port, concurrency, testDuration)
		result.ConcurrencyTests[concurrency] = concResult

		fmt.Printf("    ✅ RPS: %.1f, Latency: %v, Errors: %.1f%%\n",
			concResult.RequestsPerSec,
			concResult.AvgLatency.Round(time.Microsecond),
			concResult.ErrorRate)

		// Accept higher error rates for high concurrency
		errorThreshold := 10.0
		if concurrency > 25 {
			errorThreshold = 25.0
		}

		if concResult.ErrorRate < errorThreshold {
			totalRPS += concResult.RequestsPerSec
			totalLatency += concResult.AvgLatency
			validTests++
		}

		runtime.GC()
	}

	if validTests > 0 {
		result.RequestsPerSec = totalRPS / float64(validTests)
		result.AvgLatency = totalLatency / time.Duration(validTests)
	}

	memAfter := getMemoryUsage()
	goroutinesAfter := runtime.NumGoroutine()

	result.MemoryUsage = memAfter - memBefore
	result.GoroutineCount = goroutinesAfter - goroutinesBefore

	if bestResult := result.ConcurrencyTests[10]; bestResult != nil {
		result.TotalRequests = int64(bestResult.RequestsPerSec * testDuration.Seconds())
		result.SuccessRequests = result.TotalRequests - int64(float64(result.TotalRequests)*bestResult.ErrorRate/100)
		result.FailedRequests = result.TotalRequests - result.SuccessRequests
		result.P95Latency = bestResult.P95Latency
	}

	fmt.Printf("  ✅ %s completed!\n\n", framework)

	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	return result
}

// 📋 ENHANCED RESULTS DISPLAY
func printEnhancedResults(results []*BenchmarkResult) {
	fmt.Println("\n🏆 GO FRAMEWORK BENCHMARK RESULTS")
	fmt.Println("=" + strings.Repeat("=", 50))

	// Filter valid results
	validResults := []*BenchmarkResult{}
	for _, result := range results {
		if result != nil && result.RequestsPerSec > 0 {
			validResults = append(validResults, result)
		}
	}

	if len(validResults) == 0 {
		fmt.Println("❌ No valid results to display")
		return
	}

	// Sort by performance
	sort.Slice(validResults, func(i, j int) bool {
		return validResults[i].RequestsPerSec > validResults[j].RequestsPerSec
	})

	fmt.Println("\n📈 OVERALL PERFORMANCE RANKING:")
	for i, result := range validResults {
		fmt.Printf("%d. %-10s - %7.1f req/sec (latency: %v)\n",
			i+1, result.Framework, result.RequestsPerSec,
			result.AvgLatency.Round(time.Microsecond))
	}

	fmt.Println("\n📊 DETAILED METRICS:")
	fmt.Printf("%-10s %-12s %-15s %-15s %-12s\n",
		"Framework", "RPS", "Avg Latency", "P95 Latency", "Memory (KB)")
	fmt.Println(strings.Repeat("-", 70))

	for _, result := range validResults {
		fmt.Printf("%-10s %-12.1f %-15v %-15v %-12d\n",
			result.Framework,
			result.RequestsPerSec,
			result.AvgLatency.Round(time.Microsecond),
			result.P95Latency.Round(time.Microsecond),
			result.MemoryUsage/1024,
		)
	}

	// Concurrency breakdown
	fmt.Println("\n🔥 CONCURRENCY PERFORMANCE:")
	for _, concurrency := range concurrencyLevels {
		fmt.Printf("\n--- Concurrency: %d ---\n", concurrency)

		type concResult struct {
			framework string
			result    *ConcurrencyResult
		}
		var concResults []concResult

		for _, result := range validResults {
			if concRes, exists := result.ConcurrencyTests[concurrency]; exists {
				concResults = append(concResults, concResult{result.Framework, concRes})
			}
		}

		sort.Slice(concResults, func(i, j int) bool {
			return concResults[i].result.RequestsPerSec > concResults[j].result.RequestsPerSec
		})

		for i, cr := range concResults {
			fmt.Printf("%d. %-10s - %6.1f req/sec (errors: %.1f%%)\n",
				i+1, cr.framework, cr.result.RequestsPerSec, cr.result.ErrorRate)
		}
	}

	if len(validResults) > 0 {
		winner := validResults[0]
		fmt.Printf("\n🥇 WINNER: %s with %.1f req/sec!\n", winner.Framework, winner.RequestsPerSec)
	}
}

func main() {
	fmt.Println("🚀 GO FRAMEWORK PERFORMANCE BENCHMARK")
	fmt.Println("Testing: Fiber, Echo, Gin, TurboGo")
	fmt.Printf("⚡ Duration: %v, Concurrency: %v\n", testDuration, concurrencyLevels)
	fmt.Println("🔥 No artificial rate limiting - full speed ahead!")
	fmt.Println(strings.Repeat("=", 50))

	// Use more cores for better performance
	maxProcs := runtime.NumCPU()
	runtime.GOMAXPROCS(maxProcs)
	fmt.Printf("🖥️  Using all %d CPU cores\n\n", maxProcs)

	var results []*BenchmarkResult

	frameworks := []struct {
		name  string
		port  int
		setup func(int)
	}{
		{"Fiber", basePort, func(port int) { setupRobustFiber(port) }},
		{"Echo", basePort + 1, func(port int) { setupRobustEcho(port) }},
		{"Gin", basePort + 2, func(port int) { setupRobustGin(port) }},
		{"TurboGo", basePort + 3, func(port int) { setupRobustTurboGo(port) }},
	}

	for i, fw := range frameworks {
		if i > 0 {
			fmt.Printf("😴 System cooldown...\n")
			runtime.GC()
			time.Sleep(2 * time.Second)
		}

		fmt.Printf("🚀 Starting %s server on port %d...\n", fw.name, fw.port)
		fw.setup(fw.port)
		time.Sleep(500 * time.Millisecond)

		result := runRobustBenchmark(fw.name, fw.port)
		results = append(results, result)
	}

	printEnhancedResults(results)

	// Save results
	if len(results) > 0 {
		data, _ := json.MarshalIndent(results, "", "  ")
		filename := fmt.Sprintf("benchmark_results_%s.json", time.Now().Format("15-04-05"))
		os.WriteFile(filename, data, 0644)
		fmt.Printf("\n💾 Results saved to: %s\n", filename)
	}

	fmt.Println("\n🎉 Benchmark completed successfully!")
}
