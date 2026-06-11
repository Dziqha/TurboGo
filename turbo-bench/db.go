package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
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
		p, err = pgxpool.New(context.Background(), connStr)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		p.Config().MaxConns = 64
		pool = p
		return
	}
	panic("failed to create database pool: " + err.Error())
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
