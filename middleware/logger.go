package middleware

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Dziqha/TurboGo/core"
	"github.com/fatih/color"
	"github.com/valyala/fasthttp"
)

type LoggerConfig struct {
	Output        io.Writer
	DisableColors bool
	TimeFormat    string
}

func Logger(config ...LoggerConfig) core.Handler {
	cfg := LoggerConfig{TimeFormat: "15:04:05"}
	if len(config) > 0 {
		if config[0].Output != nil {
			cfg.Output = config[0].Output
		}
		if config[0].DisableColors {
			cfg.DisableColors = true
		}
		if config[0].TimeFormat != "" {
			cfg.TimeFormat = config[0].TimeFormat
		}
	}

	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}
	timeFormat := cfg.TimeFormat

	return func(c *core.Context) {
		c.Next()

		if core.DisableLogger {
			return
		}

		status := c.Ctx.Response.StatusCode()
		if status == 0 {
			if c.Aborted() {
				status = fasthttp.StatusUnauthorized
			} else {
				status = fasthttp.StatusOK
			}
			c.Ctx.SetStatusCode(status)
		}

		method := string(c.Ctx.Method())
		path := string(c.Ctx.Path())
		timestamp := time.Now().Format(timeFormat)

		var line string
		if cfg.DisableColors {
			line = fmt.Sprintf("🌀 TurboGo [%s] %s %s [%d]\n", timestamp, method, path, status)
		} else {
			var statusColor *color.Color
			switch {
			case status >= 200 && status < 300:
				statusColor = color.New(color.FgGreen)
			case status >= 300 && status < 400:
				statusColor = color.New(color.FgCyan)
			case status >= 400 && status < 500:
				statusColor = color.New(color.FgYellow)
			default:
				statusColor = color.New(color.FgRed)
			}

			line = fmt.Sprintf("🌀 TurboGo [%s] %s %s [%s]\n",
				color.New(color.FgBlue).Sprint(timestamp),
				color.New(color.FgMagenta).Sprint(method),
				path,
				statusColor.Sprint(status),
			)
		}

		fmt.Fprint(out, line)
	}
}