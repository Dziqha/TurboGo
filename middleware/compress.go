package middleware

import (
	"strings"

	"github.com/Dziqha/TurboGo/core"
	"github.com/valyala/fasthttp"
)

var compressiblePrefixes = []string{
	"text/",
	"application/json",
	"application/javascript",
	"application/xml",
	"application/xhtml+xml",
	"application/rss+xml",
	"application/atom+xml",
	"application/ld+json",
	"application/manifest+json",
	"application/vnd.api+json",
	"application/graphql+json",
	"image/svg+xml",
	"font/ttf",
	"font/otf",
	"font/woff",
	"font/woff2",
}

func Compress(level int) core.Handler {
	if level == 0 {
		level = fasthttp.CompressDefaultCompression
	}

	return func(c *core.Context) {
		c.Next()

		resp := &c.Ctx.Response
		if c.Aborted() || resp.BodyStream() != nil {
			return
		}
		status := resp.StatusCode()
		if status < 200 || status == fasthttp.StatusNoContent || status == fasthttp.StatusNotModified {
			return
		}
		if len(resp.Header.Peek("Content-Encoding")) > 0 {
			return
		}

		body := resp.Body()
		if len(body) < 32 {
			return
		}
		if !isCompressible(string(resp.Header.ContentType())) {
			return
		}

		req := &c.Ctx.Request
		var compressed []byte
		var encoding string
		switch {
		case req.Header.HasAcceptEncodingBytes([]byte("br")):
			compressed = fasthttp.AppendBrotliBytesLevel(nil, body, level)
			encoding = "br"
		case req.Header.HasAcceptEncodingBytes([]byte("gzip")):
			compressed = fasthttp.AppendGzipBytesLevel(nil, body, level)
			encoding = "gzip"
		default:
			return
		}

		resp.SetBodyRaw(compressed)
		resp.Header.Set("Content-Encoding", encoding)
		resp.Header.Set("Vary", "Accept-Encoding")
		resp.Header.Del("Content-Length")
	}
}

func isCompressible(contentType string) bool {
	ct := strings.ToLower(contentType)
	for _, prefix := range compressiblePrefixes {
		if strings.HasPrefix(ct, prefix) {
			return true
		}
	}
	return false
}