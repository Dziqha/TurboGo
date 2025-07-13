# 🌀 TurboGo — High Performance Middleware-First Go Framework

[![Go](https://img.shields.io/badge/Go-1.22-blue)](https://go.dev)
[![Benchmarks](https://img.shields.io/badge/Benchmarks-PASS-brightgreen)]()
[![Coverage](https://img.shields.io/badge/Coverage-ComingSoon-yellow)]()

**TurboGo** is a blazing-fast, middleware-first, and event-driven web framework built with Go — inspired by Express, but optimized for high concurrency, clean extensibility, and developer control.

---

## 📁 Project Structure

```bash

turbogo/
├── templates/          # CLI template & generator
├── core/               # HTTP context, router, handler base
├── internal/           # In-memory engines for pubsub, queue, cache, etc
│   ├── cache/          # Redis-like in-memory cache engine
│   ├── pubsub/         # Kafka-style pub/sub with topic fanout
│   ├── queue/          # Simple async job/task queue
│   └── concurrency/    # Goroutine pool, async control, lock/mutex helpers
├── middleware/         # Built-in middlewares: logger, auth, recovery
├── examples/           # Usage examples with main.go
├── test/               # Unit tests and benchmarks
├── app.go              # TurboGo core application entry
├── makefile            # Build, test, benchmark automation
└── README.md

```

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

## ⚙️ Features

- ✅ **Middleware-first** — use `.Use()` like Express.js
- ⚡ **Ultra-fast** router & context engine
- 🔄 **Built-in async engines** (PubSub, Queue, Cache)
- 🧠 **Extensible** and clean internal architecture
- 🔐 **Optional middleware**: Auth, Logger, Recovery, Auto-cache
- 🧪 Benchmark & unit-test ready
- 🛠️ CLI generator: `create-turbogo` for instant project scaffolding

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

## 🚀 CLI: Create TurboGo App

Install the TurboGo CLI:

```bash
npx create-turbogo myapp
```

```bash
cd myapp

go run .
```

## 🚀 Manual Instalation TurboGo

```bash
go get github.com/Dziqha/TurboGo
```


---

## 📦 Layer Breakdown

| Layer           | Location                                | Description                                 |
| --------------- | --------------------------------------- | ------------------------------------------- |
| **Routing**     | `core/routing.go`                       | Lightweight HTTP router with method support |
| **Middleware**  | `middleware/logger.go`, `auth.go`, etc. | Plug-and-play middleware support            |
| **Auto Cache**  | `middleware/cache.go`                   | Automatic path-based caching layer          |
| **Handlers**    | `internal/controller/`                  | Developer-defined business logic            |
| **Engines**     | `internal/pubsub/`, `queue/`, `cache/`  | Event engines without external dependencies |
| **Concurrency** | `internal/concurrency/`                 | Goroutine pooling, async, and locking utils |

---

## 📊 Benchmark Summary

Run with:

```bash
make bench
```

| Benchmark                       | Time (ns/op) | Mem  | Alloc | Status |
| ------------------------------- | ------------ | ---- | ----- | ------ |
| `BenchmarkPubSub_1000Messages`  | \~265        | 249B | 4x    | ✅      |
| `BenchmarkTaskQueue_1000Tasks`  | \~0.02       | 0B   | 0     | ✅      |
| `BenchmarkTaskQueue_WithDelay`  | \~0.17       | 0B   | 0     | ✅      |
| `BenchmarkTaskQueue_CPUProfile` | \~592        | 4B   | 1x    | ✅      |

---

## 🧰 TurboGo CLI (`create-turbogo`)

> Scaffold TurboGo apps instantly via CLI.

```bash
create-turbogo myapp
```

Prompted features:

* ✅ Controller name
* 🔐 Enable dummy Auth
* 📁 Structure auto-generated

---

## ✅ Ideal For

* ⚙️ Microservices / REST APIs
* 🚚 Background jobs / task queues
* 📡 Event-driven systems (pub/sub pipelines)
* 🧪 High-performance concurrent services

---

## ❤️ About

TurboGo is handcrafted with performance, simplicity, and extensibility in mind — empowering developers to build Go web backends without the bloat.

---

**Ready to go fast? Build with TurboGo.** 🌀
Give it a ⭐ on GitHub if you like it!

---

> Built with love by [@dziqha](https://github.com/dziqha)