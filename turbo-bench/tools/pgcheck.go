//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	connStr := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		fmt.Printf("POOL ERROR: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM World").Scan(&count)
	if err != nil {
		fmt.Printf("QUERY ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("COUNT: %d\n", count)
}
