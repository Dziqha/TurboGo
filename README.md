
# TurboGo Framework Architecture

TurboGo is a middleware-first, event-driven Go backend framework focused on performance and developer experience. Below is the detailed architecture flow.

---

## 🧭 Request Lifecycle

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

## 📦 Layer Breakdown

| Layer/Komponen              | File / Modul                         | Deskripsi                                                                 |
|-----------------------------|--------------------------------------|--------------------------------------------------------------------------|
| **Client**                  | Browser, curl, postman               | Mengirim request ke server                                               |
| **HTTP Router**             | `core/routing.go`                    | Menyediakan `.Get`, `.Post`, dll                                         |
| **Middleware Pipeline**     | `middleware/*.go`                    | Logger, Cache, Auth, Recovery                                            |
| **Redis Auto-Cache Layer**  | `middleware/cache.go`                | Otomatis menyimpan dan mengambil cache berdasarkan path atau key         |
| **Handler Logic**           | `handlers/*.go`                      | Fungsi-fungsi buatan developer                                           |
| **Internal Task Engine**    | `core/context.go` + `internal/*`     | Pipeline event-driven (Kafka, RabbitMQ, Redis, dll)                      |
| **Concurrency Tools**       | `internal/concurrency/`              | Goroutine, channel, mutex bawaan Go                                      |
| **Final Response**          | `ctx.JSON(...)`                      | Menyimpan ke cache (jika aktif) + kirim respons ke user                  |

---

## 🧱 Modular File Structure (v1)


```
turbogo/
├── cmd/
│   └── root.go
├── core/
│   ├── app.go
│   ├── context.go
│   ├── routing.go
│   └── config.go
├── internal/
│   ├── redis/
│   │   ├── inmem.go
│   │   └── persist.go
│   ├── kafka/
│   │   ├── pubsub.go
│   │   └── persist.go
│   ├── rabbitmq/
│   │   ├── taskqueue.go
│   │   └── persist.go
│   └── concurrency/
│       ├── async.go
│       └── mutex.go
├── middleware/
│   ├── cache.go
│   ├── logger.go
│   ├── recover.go
│   └── auth.go
├── pkg/
│   └── jsonutil/
├── examples/
│   └── main.go
├── test/
│   └── all_test.go
└── README.md
```

---

## ✅ Summary

TurboGo menggabungkan kekuatan:

- 🧠 Clean Architecture
- ⚙️ Middleware-first Design
- 🚀 Event-Driven Flow
- ⚡ Built-in Concurrency (goroutine, channel)
- 📦 Auto Redis Caching
- 📮 Kafka + RabbitMQ Integration

```
💡 Goal: One route = one powerful pipeline, without boilerplate.
```

---

```
Created with ❤️ by TurboGo
```
