package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"io"
	"os"
)

func openDatabase(diagnostics io.Writer) (*sql.DB, int) {
	dsn := os.Getenv("LEDGER_DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(diagnostics, "set LEDGER_DATABASE_URL")
		return nil, 2
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintln(diagnostics, "invalid PostgreSQL configuration")
		return nil, 1
	}
	return db, 0
}
