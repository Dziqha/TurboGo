package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

func getDBConnStr() string {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		host := getEnv("DBHOST", "localhost")
		port := getEnv("DBPORT", "5432")
		user := getEnv("DBUSER", "benchmark")
		pass := getEnv("DBPASS", "benchmark")
		dbname := getEnv("DBNAME", "hello_world")
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, dbname)
	}
	return connStr
}

func initDB() {
	connStr := getDBConnStr()

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

func fetchWorlds(ids []int32) ([]World, error) {
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

func batchExec(worlds []World) {
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
	pool.Exec(ctx, q.String(), args...)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
