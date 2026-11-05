# TurboGo

[![Go](https://img.shields.io/badge/Go-1.25-blue)](https://go.dev)
[![Benchmarks](https://img.shields.io/badge/Benchmarks-PASS-brightgreen)]()
[![Go Report Card](https://goreportcard.com/badge/github.com/Dziqha/TurboGo)](https://goreportcard.com/report/github.com/Dziqha/TurboGo)
[![Coverage](https://img.shields.io/badge/Coverage-51.9%25-yellow)](https://github.com/Dziqha/TurboGo/actions)
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
- Test helpers: `app.Test(req)` runs requests in-process, no port needed
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

### TurboGo CLI (`npx create-turbogo@latest`)

> Scaffold TurboGo apps instantly via CLI.

```bash
npx create-turbogo@latest myapp
```

Interactive prompts:

* ✅ Feature name (e.g. user, auth, order)

* 📁 Feature-based project structure generated automatically

* ⚙️ Ready-to-extend architecture out of the box

> ⚠️ We intentionally use Node.js to overcome limitations of Go's CLI tooling, especially for rich interactive workflows.

### Running TurboGo Example


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

Measured on a 6C/12T Ryzen 5 5600H with Go 1.26, `wrk` on the same host,
round-robin reps, median of 5. Method, spread and caveats:
[Benchmarks docs](docs/content/docs/benchmarks.mdx).

| Workload | TurboGo | fasthttp | Fiber v3 | Gin | Echo |
|---|---|---|---|---|---|
| `GET /user/42`, c=200, t=6 | 386,102 | 393,594 | 378,991 | — | — |
| `GET /user/42`, c=512, t=12 | 389,963 | 426,374 | 414,118 | 259,817 | 249,975 |
| In-process, per request | **155 ns** | 166 ns (bare path+switch) | 308 ns | — | — |

TurboGo is a thin layer over fasthttp: at thread-matched concurrency it is
within noise of fasthttp and Fiber; at oversubscribed concurrency every
fasthttp-based server lands inside the ±15% spread. What the framework adds is
zero-allocation routing and built-in engines, not a different HTTP engine.

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