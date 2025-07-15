# 🌀 TurboGo — High Performance Middleware-First Go Framework

[![Go](https://img.shields.io/badge/Go-1.24-blue)](https://go.dev)
[![Benchmarks](https://img.shields.io/badge/Benchmarks-PASS-brightgreen)]()
[![Coverage](https://img.shields.io/badge/Coverage-ComingSoon-yellow)]()
[![Contributions](https://img.shields.io/badge/Contributions-welcome-blueviolet)](https://github.com/Dziqha/TurboGo/discussions)
[![License](https://img.shields.io/github/license/Dziqha/TurboGo)](./LICENSE)
[![Last Commit](https://img.shields.io/github/last-commit/Dziqha/TurboGo)](https://github.com/Dziqha/TurboGo/commits)
[![Issues](https://img.shields.io/github/issues/Dziqha/TurboGo)](https://github.com/Dziqha/TurboGo/issues)

**TurboGo** is a blazing-fast, middleware-first, and event-driven web framework built with Go — inspired by Express, but optimized for high concurrency, clean extensibility, and developer control.

**TurboGo Key Features are :**

- Middleware-first — use `.Use()` like Express.js
- Ultra-fast router & context engine
- Built-in async engines (PubSub, Queue, Cache)
- Extensible and clean internal architecture
- Optional middleware: Auth, Logger, Recovery, Auto-cache
- CLI generator: `npx create-turbogo` for instant project scaffolding


---

## 🧭 Request Lifecycle Overview

```
                           ┌────────────┐
                           │   Client   │
                           └────┬───────┘
                                │
                                ▼
                        ┌───────────────┐
                        │  HTTP Router  │  ← core/routing.go
                        └─────┬─────────┘
                              ▼
              ┌────────────────────────────────┐
              │  Turbo Middleware Pipeline      │ ← middleware/logger.go, auth.go, etc.
              └────────┬───────────────────────┘
                       ▼
        ┌────────────────────────────────────────────┐
        │ Redis Auto-Cache Layer (Check & Set)        │ ← middleware/cache.go
        └────────┬──────────────────────────────┬─────┘
                 ▼                              ▼
          Cache Hit → Return JSON       Cache Miss → Proceed
                                                 │
                                                 ▼
                           ┌──────────────────────────────┐
                           │     Handler Logic (Dev)      │ ← developer handler: func(ctx *Context)
                           └──────────────┬───────────────┘
                                          ▼
                     ┌────────────────────────────────────────────┐
                     │       Embedded Infrastructure Engine       │ ← core/context.go
                     └──────┬────────────┬────────────┬───────────┘
                            ▼            ▼            ▼
                         Redis         Kafka       RabbitMQ
                      (inmem.go)   (pubsub.go)   (taskqueue.go)
                            ▼            ▼            ▼
                         persist       persist       persist
                          (.json)       (.log)        (.log)
                                ▼
                        Response + Cache Set
```

---

## 🔐 Example: Auth Middleware

Use the built-in auth middleware with environment variable `AUTH_SECRET`:

```go
app.Use(middleware.Auth(os.Getenv("AUTH_SECRET")))
```

Set your secret in `.env`:

```bash
export AUTH_SECRET=supersecurekey123
```

---

## 🚀 Getting started

###  Manual Installation TurboGo

```bash
go get github.com/Dziqha/TurboGo
```

### TurboGo CLI (`npx create-turbogo`)

> Scaffold TurboGo apps instantly via CLI.

```bash
npx create-turbogo myapp
```

Prompted features:

* ✅ Controller name
* 📁 Structure auto-generated

> ⚠️ We intentionally use Node.js to overcome limitations of Go's CLI tooling, especially for rich interactive workflows.

### Running TurboGo Example

> Gin requires Go version `1.23 or above.`

```go
package main

import (
	"github.com/Dziqha/TurboGo"
	"github.com/Dziqha/TurboGo/core"
)

func main() {
	app := TurboGo.New()

	app.Get("/", func(c *core.Context) {
		c.Text(200, "Hello from TurboGo!")
	})

	app.RunServer(":8080")
}
```
To run the code, use the `go run` command, like:

```bash
go run example.go
```

> Your TurboGo app will be available at http://localhost:8080
---

## 🧪 Benchmark Overview

| Benchmark Name | Iterations | Time per Operation | Memory per Operation | Allocations per Operation |
|---|---|---|---|---|
| BenchmarkPubSub_1000Messages-12 | 3,941,121 | 284.3 ns/op | 249 B/op | 4 allocs/op |
| BenchmarkTaskQueue_1000Tasks-12 | 4,266,667 | 297.1 ns/op | 4 B/op | 1 allocs/op |
| BenchmarkTaskQueue_WithDelay-12 | 1,069 | 1,124,821 ns/op | 252 B/op | 4 allocs/op |
| BenchmarkTaskQueue_CPUProfile-12 | 1,883,364 | 652.6 ns/op | 4 B/op | 1 allocs/op |
| BenchmarkCPUPrint-12 | 1,000,000,000 | 0.4956 ns/op | 0 B/op | 0 allocs/op |
| BenchmarkTaskQueue_Print-12 | 1,809,789 | 652.3 ns/op | 4 B/op | 1 allocs/op |
| BenchmarkTaskQueue_Parallel-12 | 28,087,125 | 46.83 ns/op | 18 B/op | 1 allocs/op |
| BenchmarkTaskQueue_DelayRetry-12 | 24,147,055 | 48.40 ns/op | 19 B/op | 1 allocs/op |
| BenchmarkTaskQueue_WorkerPool-12 | 4,324,900 | 306.5 ns/op | 4 B/op | 1 allocs/op |
| BenchmarkTaskQueue_RateLimit-12 | 2,259 | 533,127 ns/op | 3 B/op | 1 allocs/op |

> Benchmarked on Windows `amd64, Ryzen 5 5600H, Go 1.24.` ⚠️ Results may vary—run on idle system for accuracy.
---

## 🤝 Contributing

Contributions are welcome — from fixing typos to suggesting ideas or building features. TurboGo grows through small steps, open discussions, and shared curiosity. Join the conversation on [Discussions](https://github.com/Dziqha/TurboGo/discussions).

## ❤️ About

TurboGo is handcrafted with performance, simplicity, and extensibility in mind — empowering developers to build Go web backends without the bloat.

**Ready to go fast? Build with TurboGo.** 🌀
Give it a ⭐ on GitHub if you like it!

---

## 📄 License

This project is licensed under the [MIT License](./LICENSE)


> Built with love by [@dziqha](https://github.com/dziqha)