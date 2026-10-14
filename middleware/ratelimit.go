package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

type RateLimitOptions struct {
	TrustForwarded bool
}

type rateBucket struct {
	tokens     float64
	lastRefill time.Time
}

type rateLimiter struct {
	limit          float64
	rate           float64
	window         time.Duration
	trustForwarded bool

	mu        sync.Mutex
	clients   map[string]*rateBucket
	lastClean time.Time
}

func RateLimit(limit int, window time.Duration, opts ...RateLimitOptions) core.Handler {
	rl := &rateLimiter{
		limit:     float64(limit),
		rate:      float64(limit) / window.Seconds(),
		window:    window,
		clients:   make(map[string]*rateBucket),
		lastClean: time.Now(),
	}
	if len(opts) > 0 {
		rl.trustForwarded = opts[0].TrustForwarded
	}

	return func(c *core.Context) {
		key := rl.clientKey(c)
		now := time.Now()

		rl.mu.Lock()
		b, ok := rl.clients[key]
		if !ok {
			b = &rateBucket{tokens: rl.limit, lastRefill: now}
			rl.clients[key] = b
		}

		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens = min(rl.limit, b.tokens+elapsed*rl.rate)
		b.lastRefill = now

		allowed := b.tokens >= 1
		if allowed {
			b.tokens--
		}

		if now.Sub(rl.lastClean) > rl.window {
			for k, v := range rl.clients {
				if now.Sub(v.lastRefill) > rl.window {
					delete(rl.clients, k)
				}
			}
			rl.lastClean = now
		}
		rl.mu.Unlock()

		if !allowed {
			c.Ctx.Response.Header.Set("Retry-After", strconv.Itoa(int(rl.window.Seconds())))
			c.Ctx.SetStatusCode(fasthttp.StatusTooManyRequests)
			c.Ctx.SetBodyString("rate limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (rl *rateLimiter) clientKey(c *core.Context) string {
	if rl.trustForwarded {
		if xff := c.Ctx.Request.Header.Peek("X-Forwarded-For"); len(xff) > 0 {
			for i, b := range xff {
				if b == ',' {
					return string(xff[:i])
				}
			}
			return string(xff)
		}
	}
	return c.Ctx.RemoteIP().String()
}