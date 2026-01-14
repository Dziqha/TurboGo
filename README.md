# TurboGo

[![Go](https://img.shields.io/badge/Go-1.24-blue)](https://go.dev)
[![Benchmarks](https://img.shields.io/badge/Benchmarks-PASS-brightgreen)]()
[![Go Report Card](https://goreportcard.com/badge/github.com/Dziqha/TurboGo)](https://goreportcard.com/report/github.com/Dziqha/TurboGo)
[![Coverage](https://img.shields.io/badge/Coverage-ComingSoon-yellow)]()
[![Contributions](https://img.shields.io/badge/Contributions-welcome-blueviolet)](https://github.com/Dziqha/TurboGo/discussions)
[![License](https://img.shields.io/github/license/Dziqha/TurboGo)](./LICENSE)
[![Last Commit](https://img.shields.io/github/last-commit/Dziqha/TurboGo)](https://github.com/Dziqha/TurboGo/commits)
[![Issues](https://img.shields.io/github/issues/Dziqha/TurboGo)](https://github.com/Dziqha/TurboGo/issues)

<img src="./docs/public/images/icon.png" alt="TurboGo Banner" width="100%"/>

**TurboGo** employs a `Tiered Zero-Copy Routing (TZCR)` system that categorizes routes into three levels—static, parametric, and wildcard—for optimal performance. Each route is precompiled and stored in a cache-aware structure, enabling fast, zero-allocation matching. Middleware is executed through efficient handler chaining, and route grouping allows modular design with custom prefixes and middlewares. This approach makes TurboGo ideal for high-performance applications without sacrificing flexibility.

**TurboGo Key Features are :**

- Middleware-first — use `.Use()`
- Ultra-fast router & context engine
- Built-in async engines (PubSub, Queue, Cache)
- Extensible and clean internal architecture
- Optional middleware: Auth, Logger, Recovery
- CLI generator: `npx create-turbogo` for instant project scaffolding

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

| Benchmark                       | Iterations    | Time/op   | Bytes/op | Allocs/op |
| ------------------------------- | ------------- | --------- | -------- | --------- |
| `BenchmarkPubSub_1000Messages`  | 4,181,076     | 270.6 ns  | 249 B    | 4         |
| `BenchmarkTaskQueue_1000Tasks`  | 4,182,391     | 272.4 ns  | 4 B      | 1         |
| `BenchmarkTaskQueue_WithDelay`  | 1,071         | 1.107 ms  | 252 B    | 4         |
| `BenchmarkTaskQueue_CPUProfile` | 1,889,114     | 644.6 ns  | 4 B      | 1         |
| `BenchmarkCPUPrint`             | 1,000,000,000 | 0.4827 ns | 0 B      | 0         |
| `BenchmarkTaskQueue_Print`      | 1,880,923     | 638.5 ns  | 4 B      | 1         |
| `BenchmarkTaskQueue_Parallel`   | 27,765,244    | 43.82 ns  | 18 B     | 1         |
| `BenchmarkTaskQueue_DelayRetry` | 26,241,208    | 45.48 ns  | 19 B     | 1         |
| `BenchmarkTaskQueue_WorkerPool` | 4,241,274     | 295.7 ns  | 4 B      | 1         |
| `BenchmarkTaskQueue_RateLimit`  | 2,211         | 529.5 µs  | 3 B      | 1         |


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