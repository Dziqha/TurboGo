//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	connStr := os.Getenv("DATABASE_URL")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Printf("POOL ERROR: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		fmt.Printf("ACQUIRE ERROR: %v\n", err)
		os.Exit(1)
	}
	defer conn.Release()

	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM World").Scan(&count)
	if err != nil {
		fmt.Printf("QUERY ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("COUNT: %d\n", count)
}
