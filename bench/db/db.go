package db

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pool  *pgxpool.Pool
	idsMu sync.Mutex
)

type World struct {
	ID           int32 `json:"id"`
	RandomNumber int32 `json:"randomNumber"`
}

type Fortune struct {
	ID      int32  `json:"id"`
	Message string `json:"message"`
}

func Init() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://benchmark:benchmark@localhost:5432/hello_world"
	}

	var p *pgxpool.Pool
	var err error
	for i := 0; i < 30; i++ {
		cfg, err := pgxpool.ParseConfig(connStr)
		if err == nil {
			cfg.MaxConns = 12
			p, err = pgxpool.NewWithConfig(context.Background(), cfg)
		}
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		pool = p
		return
	}
	panic("failed to create database pool: " + err.Error())
}

func Ready() bool {
	return pool != nil
}

func RandID() int32 {
	return int32(rand.IntN(10000) + 1)
}

func RandIDs(n int) []int32 {
	idsMu.Lock()
	defer idsMu.Unlock()
	s := make([]int32, n)
	for i := range s {
		s[i] = int32(rand.IntN(10000) + 1)
	}
	return s
}

func FetchOne(id int32) (*World, error) {
	var w World
	err := pool.QueryRow(context.Background(),
		"SELECT id, randomNumber FROM World WHERE id = $1", id,
	).Scan(&w.ID, &w.RandomNumber)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func FetchMany(ids []int32) ([]World, error) {
	rows, err := pool.Query(context.Background(),
		"SELECT id, randomNumber FROM World WHERE id = ANY($1)", ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[int32]int32, len(ids))
	for rows.Next() {
		var id, rn int32
		if err := rows.Scan(&id, &rn); err != nil {
			return nil, err
		}
		m[id] = rn
	}

	worlds := make([]World, len(ids))
	for i, id := range ids {
		worlds[i] = World{ID: id, RandomNumber: m[id]}
	}
	return worlds, nil
}

func UpdateAll(worlds []World) error {
	ctx := context.Background()
	var q strings.Builder
	q.WriteString("UPDATE World SET randomNumber = CASE id")
	args := make([]any, 0, len(worlds)*2)
	for i, w := range worlds {
		q.WriteString(fmt.Sprintf(" WHEN $%d::int THEN $%d::int", i*2+1, i*2+2))
		args = append(args, w.ID, w.RandomNumber)
	}
	q.WriteString(" END WHERE id IN (")
	for i := range worlds {
		if i > 0 {
			q.WriteString(",")
		}
		q.WriteString(fmt.Sprintf("$%d::int", i*2+1))
	}
	q.WriteString(")")
	_, err := pool.Exec(ctx, q.String(), args...)
	return err
}

func AllFortunes() ([]Fortune, error) {
	rows, err := pool.Query(context.Background(),
		"SELECT id, message FROM Fortune")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fortunes := make([]Fortune, 0, 16)
	for rows.Next() {
		var f Fortune
		if err := rows.Scan(&f.ID, &f.Message); err != nil {
			return nil, err
		}
		fortunes = append(fortunes, f)
	}
	return fortunes, nil
}

func RenderFortunes(fortunes []Fortune) string {
	fortunes = append(fortunes, Fortune{
		ID:      0,
		Message: "Additional fortune added at request time.",
	})
	sort.Slice(fortunes, func(i, j int) bool {
		return fortunes[i].Message < fortunes[j].Message
	})

	var b strings.Builder
	b.Grow(2048)
	b.WriteString("<!DOCTYPE html><html><head><title>Fortunes</title></head><body><table><tr><th>id</th><th>message</th></tr>")
	for _, f := range fortunes {
		b.WriteString("<tr><td>")
		b.WriteString(fmt.Sprintf("%d", f.ID))
		b.WriteString("</td><td>")
		b.WriteString(escapeHTML(f.Message))
		b.WriteString("</td></tr>")
	}
	b.WriteString("</table></body></html>")
	return b.String()
}

var htmlEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&#39;",
)

func escapeHTML(s string) string {
	return htmlEscaper.Replace(s)
}

func ParseQueryCount(q string) int {
	if q == "" {
		return 1
	}
	n := 0
	for _, ch := range q {
		if ch < '0' || ch > '9' {
			return 1
		}
		n = n*10 + int(ch-'0')
	}
	if n < 1 {
		return 1
	}
	if n > 500 {
		return 500
	}
	return n
}