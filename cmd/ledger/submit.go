package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Vitaliy-Pikalo/compute-job-ledger/internal/jobs"
	"github.com/Vitaliy-Pikalo/compute-job-ledger/internal/risk"
	_ "github.com/lib/pq"
	"io"
	"os"
	"time"
)

func submit(args []string, out, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("submit", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	key := flags.String("request-id", "", "stable client request ID")
	seed := flags.Int64("seed", 42, "random seed")
	trials := flags.Int("trials", 1000, "number of trials")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *key == "" || *trials < 1 || *trials > risk.MaxTrials {
		fmt.Fprintln(diagnostics, "submit requires -request-id and a valid trial count")
		return 2
	}
	dsn := os.Getenv("LEDGER_DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(diagnostics, "set LEDGER_DATABASE_URL")
		return 2
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintln(diagnostics, "invalid PostgreSQL configuration")
		return 1
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id, err := jobs.Submit(ctx, db, *key, *seed, *trials)
	if errors.Is(err, jobs.ErrConflict) {
		fmt.Fprintln(diagnostics, err)
		return 3
	}
	if err != nil {
		fmt.Fprintln(diagnostics, "database submission failed; check connectivity, credentials and migrations")
		return 1
	}
	if err = json.NewEncoder(out).Encode(struct {
		ID int64 `json:"job_id"`
	}{id}); err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	return 0
}
