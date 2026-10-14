package middleware

import (
	"time"

	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

// Timeout limits the execution time of the handler chain.
//
// When the deadline is reached, a 408 response is written and the middleware
// returns immediately. The handler goroutine keeps running in the background;
// handlers must not touch the context after the deadline, as it may be
// recycled for another request.
func Timeout(d time.Duration) core.Handler {
	return func(c *core.Context) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			c.Next()
		}()

		timer := time.NewTimer(d)
		defer timer.Stop()

		select {
		case <-done:
		case <-timer.C:
			c.Abort()
			c.Ctx.SetStatusCode(fasthttp.StatusRequestTimeout)
			c.Ctx.SetBodyString("request timeout")
		}
	}
}