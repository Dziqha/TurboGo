package middleware

import (
	"github.com/Dziqha/TurboGo/core"
	"fmt"
	"time"

	"github.com/fatih/color"
)

func Logger() core.Handler {
	return func(c *core.Context) {
		start := time.Now()

		defer func() {
			duration := time.Since(start)
			ns := max(duration.Nanoseconds(), 100)

			var durationColor *color.Color
			switch {
			case ns > 10_000_000: // >10ms
				durationColor = color.New(color.FgRed)
			case ns > 1_000_000: // >1ms
				durationColor = color.New(color.FgYellow)
			default:
				durationColor = color.New(color.FgGreen)
			}

			var durationStr string
			switch {
			case ns >= 1e9:
				durationStr = fmt.Sprintf("%.3fs", float64(ns)/1e9)
			case ns >= 1e6:
				durationStr = fmt.Sprintf("%.3fms", float64(ns)/1e6)
			case ns >= 1e3:
				durationStr = fmt.Sprintf("%.3fµs", float64(ns)/1e3)
			default:
				durationStr = fmt.Sprintf("%dns", ns)
			}

			status := c.Ctx.Response.StatusCode()
			if status == 0 {
				status = 200
			}

			method := string(c.Ctx.Method())
			path := string(c.Ctx.Path())

			var statusColor *color.Color
			switch {
			case status >= 500:
				statusColor = color.New(color.FgRed)
			case status >= 400:
				statusColor = color.New(color.FgYellow)
			default:
				statusColor = color.New(color.FgGreen)
			}

			fmt.Printf("→ %-7s %-30s %s (%s)\n",
				method,
				path,
				statusColor.Sprintf("[%3d]", status),
				durationColor.Sprint(durationStr),
			)
		}()

		c.Next()
	}
}
