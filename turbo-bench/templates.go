package main

import (
	"html"
	"strings"
)

const fortuneHTMLPrefix = `<!DOCTYPE html>
<html>
<head><title>Fortunes</title></head>
<body><table><tr><th>id</th><th>message</th></tr>
`

const fortuneHTMLSuffix = `</table></body>
</html>`

func renderFortunes(fortunes []Fortune) string {
	var b strings.Builder
	b.Grow(512 + len(fortunes)*64)
	b.WriteString(fortuneHTMLPrefix)
	for _, f := range fortunes {
		b.WriteString("<tr><td>")
		b.WriteString(itoa(int(f.ID)))
		b.WriteString("</td><td>")
		b.WriteString(html.EscapeString(f.Message))
		b.WriteString("</td></tr>\n")
	}
	b.WriteString(fortuneHTMLSuffix)
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [11]byte
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
