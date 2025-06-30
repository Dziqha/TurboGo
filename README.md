# 🌀 TurboGo — High Performance Middleware-First Go Framework

[![Go](https://img.shields.io/badge/Go-1.22-blue)](https://go.dev)
[![Benchmarks](https://img.shields.io/badge/Benchmarks-PASS-brightgreen)]()
[![Coverage](https://img.shields.io/badge/Coverage-ComingSoon-yellow)]()

TurboGo adalah framework backend berbasis Go yang ringan, middleware-first, dan event-driven. Fokus utama pada kecepatan, kemudahan extensibility, dan developer experience.

---

## 📁 Project Structure

```
turbogo/
├── cmd/                # CLI commands (generate, etc)
├── core/               # HTTP context, router, logger, handler base
├── internal/           # Engine untuk cache, pubsub, queue, concurrency
│   ├── cache/          # Redis-like engine (in-memory)
│   ├── pubsub/         # Kafka-style pubsub engine
│   ├── queue/          # RabbitMQ-style task queue
│   └── concurrency/    # Utility async/goroutine/mutex helpers
├── middleware/         # Auth, Logger, Recovery, Cache layer
├── examples/           # Example main.go app
├── test/               # Benchmark & unit test
├── app.go              # Main app entry
├── makefile
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

## 🔐 Middleware: Auth Example

Gunakan `AUTH_SECRET` dari environment:

```go
app.Use(middleware.Auth(os.Getenv("AUTH_SECRET")))
```

Atur env:
```bash
export AUTH_SECRET=supersecurekey123
```

---

## 📦 Layer Breakdown

| Layer                | File                            | Deskripsi                                               |
|---------------------|----------------------------------|----------------------------------------------------------|
| Router              | `core/routing.go`                | Basic `.Get()`, `.Post()` route register                |
| Middleware          | `middleware/logger.go`, `auth.go`| Middleware pipeline (logger, auth, recovery, cache)     |
| Redis Auto-Cache    | `middleware/cache.go`            | Cek dan simpan response otomatis via path/key           |
| Handler             | `handlers/*.go`                  | Business logic dibuat oleh developer                    |
| Embedded Engines    | `internal/*`                     | TaskQueue, PubSub, dan Cache in-memory engine           |
| Concurrency Tools   | `internal/concurrency/*.go`      | Channel pool, goroutine control, mutex helper           |

---

## 📊 Benchmark Summary

> Jalankan:
```bash
make bench
```

| Benchmark                        | ns/op    | Mem | Alloc | Status |
|----------------------------------|----------|------|--------|--------|
| `BenchmarkPubSub_1000Messages`   | ~265     | 249B | 4x     | ✅     |
| `BenchmarkTaskQueue_1000Tasks`   | ~0.02    | 0B   | 0      | ✅     |
| `BenchmarkTaskQueue_WithDelay`   | ~0.17    | 0B   | 0      | ✅     |
| `BenchmarkTaskQueue_CPUProfile`  | ~592     | 4B   | 1x     | ✅     |

---

## ✅ Goals

- ⚙️ Middleware-first, seperti Express
- 📮 Mendukung Kafka, RabbitMQ tanpa import eksternal
- ⚡ Sangat cepat (sub-microsecond op)
- 🧠 Clean code & extensible
- ✅ Siap digunakan untuk proyek microservice, REST API, atau pubsub pipelines

---

```
Created with ❤️ by TurboGo
```