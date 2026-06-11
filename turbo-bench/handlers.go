package main

import (
	"context"
	"math/rand/v2"
	"sort"
	"strconv"
	"sync"

	"github.com/Dziqha/TurboGo/core"
)

var idPool = sync.Pool{
	New: func() any {
		s := make([]int32, 0, 512)
		return &s
	},
}

func randIDs(n int) []int32 {
	ptr := idPool.Get().(*[]int32)
	s := *ptr
	if cap(s) < n {
		s = make([]int32, n)
	}
	s = s[:n]
	for i := range s {
		s[i] = int32(rand.IntN(10000) + 1)
	}
	return s
}

func putIDs(s []int32) {
	s = s[:0]
	idPool.Put(&s)
}

func PlaintextHandler(c *core.Context) {
	c.Ctx.SetContentType("text/plain")
	c.Ctx.SetBodyString("Hello, World!")
}

func JSONHandler(c *core.Context) {
	c.JSON(200, Message{Message: "Hello, World!"})
}

func requireDB(c *core.Context) bool {
	if pool == nil {
		c.Ctx.SetStatusCode(503)
		c.Ctx.SetBodyString("Database not ready")
		return false
	}
	return true
}

func DBHandler(c *core.Context) {
	if !requireDB(c) {
		return
	}
	id := int32(rand.IntN(10000) + 1)

	var w World
	err := pool.QueryRow(context.Background(),
		"SELECT id, randomNumber FROM World WHERE id = $1", id,
	).Scan(&w.ID, &w.RandomNumber)
	if err != nil {
		c.Ctx.SetStatusCode(500)
		return
	}

	c.JSON(200, &w)
}

func QueriesHandler(c *core.Context) {
	if !requireDB(c) {
		return
	}
	n := parseQueryCount(c)
	if n == 0 {
		n = 1
	}

	ids := randIDs(n)
	defer putIDs(ids)

	worlds, err := fetchWorlds(ids)
	if err != nil {
		c.Ctx.SetStatusCode(500)
		return
	}

	c.JSON(200, worlds)
}

func FortunesHandler(c *core.Context) {
	if !requireDB(c) {
		return
	}
	rows, err := pool.Query(context.Background(),
		"SELECT id, message FROM Fortune")
	if err != nil {
		c.Ctx.SetStatusCode(500)
		return
	}
	defer rows.Close()

	fortunes := make([]Fortune, 0, 16)
	for rows.Next() {
		var f Fortune
		if err := rows.Scan(&f.ID, &f.Message); err != nil {
			c.Ctx.SetStatusCode(500)
			return
		}
		fortunes = append(fortunes, f)
	}

	fortunes = append(fortunes, Fortune{
		ID:      0,
		Message: "Additional fortune added at request time.",
	})

	sort.Slice(fortunes, func(i, j int) bool {
		return fortunes[i].Message < fortunes[j].Message
	})

	html := renderFortunes(fortunes)
	c.Ctx.SetContentType("text/html; charset=utf-8")
	c.Ctx.SetBodyString(html)
}

func UpdateHandler(c *core.Context) {
	if !requireDB(c) {
		return
	}
	n := parseQueryCount(c)
	if n == 0 {
		n = 1
	}

	ids := randIDs(n)
	defer putIDs(ids)

	worlds, err := fetchWorlds(ids)
	if err != nil {
		c.Ctx.SetStatusCode(500)
		return
	}

	for i := range worlds {
		worlds[i].RandomNumber = int32(rand.IntN(10000) + 1)
	}

	batchExec(worlds)

	c.JSON(200, worlds)
}

func parseQueryCount(c *core.Context) int {
	q := c.Ctx.QueryArgs().Peek("q")
	if len(q) == 0 {
		return 0
	}
	n, err := strconv.Atoi(string(q))
	if err != nil || n < 1 {
		return 1
	}
	if n > 500 {
		return 500
	}
	return n
}
