package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/lib/pq"
)

type lease struct {
	JobID     int64     `json:"job_id"`
	Seed      int64     `json:"seed"`
	Trials    int       `json:"trials"`
	Token     string    `json:"lease_token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func claim(args []string, out, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("claim", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(diagnostics, "claim accepts no arguments")
		return 2
	}
	db, code := openDatabase(diagnostics)
	if code != 0 {
		return code
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var result lease
	var response *lease
	err := db.QueryRowContext(ctx, "SELECT job_id, seed, trials, token, expires_at FROM public.claim_job()").Scan(&result.JobID, &result.Seed, &result.Trials, &result.Token, &result.ExpiresAt)
	var pgError *pq.Error
	switch {
	case err == nil:
		response = &result
	case errors.Is(err, sql.ErrNoRows): // Encode an empty poll as JSON null.
	case errors.As(err, &pgError) && pgError.Code == "42501":
		fmt.Fprintln(diagnostics, "claim permission denied")
		return 4
	default:
		fmt.Fprintln(diagnostics, "database claim failed; check connectivity, credentials and migrations")
		return 1
	}
	if err := json.NewEncoder(out).Encode(response); err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	return 0
}
