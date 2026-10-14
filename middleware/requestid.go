package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/Dziqha/TurboGo/core"
)

type RequestIDConfig struct {
	Header    string
	Length    int
	Generator func() string
}

var requestIDFallback atomic.Uint64

func RequestID(config ...RequestIDConfig) core.Handler {
	cfg := RequestIDConfig{
		Header: "X-Request-ID",
		Length: 16,
	}
	if len(config) > 0 {
		if config[0].Header != "" {
			cfg.Header = config[0].Header
		}
		if config[0].Length > 0 {
			cfg.Length = config[0].Length
		}
		if config[0].Generator != nil {
			cfg.Generator = config[0].Generator
		}
	}

	gen := cfg.Generator
	if gen == nil {
		gen = func() string {
			b := make([]byte, cfg.Length)
			if _, err := rand.Read(b); err == nil {
				return hex.EncodeToString(b)
			}
			return fmt.Sprintf("%d-%d", os.Getpid(), requestIDFallback.Add(1))
		}
	}

	return func(c *core.Context) {
		id := string(c.Ctx.Request.Header.Peek(cfg.Header))
		if id == "" {
			id = gen()
		}
		c.SetSession("request_id", id)
		c.Ctx.Response.Header.Set(cfg.Header, id)
		c.Next()
	}
}