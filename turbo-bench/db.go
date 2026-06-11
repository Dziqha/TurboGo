package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

func initDB() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		host := getEnv("DBHOST", "localhost")
		port := getEnv("DBPORT", "5432")
		user := getEnv("DBUSER", "benchmark")
		pass := getEnv("DBPASS", "benchmark")
		dbname := getEnv("DBNAME", "hello_world")
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, dbname)
	}

	var err error
	pool, err = pgxpool.New(context.Background(), connStr)
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	pool.Config().MaxConns = 64
}

func batchExec(worlds []World) {
	ctx := context.Background()
	batch := &pgx.Batch{}
	for _, w := range worlds {
		batch.Queue("UPDATE World SET randomNumber = $1 WHERE id = $2",
			w.RandomNumber, w.ID)
	}
	br := pool.SendBatch(ctx, batch)
	br.Close()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
