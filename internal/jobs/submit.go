// Package jobs persists compute job requests.
package jobs

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Vitaliy-Pikalo/compute-job-ledger/internal/risk"
)

var ErrConflict = errors.New("request ID already belongs to different inputs")

// InputHash commits to canonical ASCII decimal integers and a workload version.
func InputHash(seed int64, trials int) [32]byte {
	return sha256.Sum256([]byte(fmt.Sprintf("risk/v1\n%d\n%d\n", seed, trials)))
}

var ErrDenied = errors.New("submission denied")

func Submit(ctx context.Context, db *sql.DB, key string, seed int64, trials int) (int64, error) {
	if key == "" || trials < 1 || trials > risk.MaxTrials {
		return 0, errors.New("expected a request ID and a valid trial count")
	}
	// One autocommitted statement persists the decision even when access is denied.
	var outcome string
	var id sql.NullInt64
	err := db.QueryRowContext(ctx, "SELECT outcome, job_id FROM public.submit_risk($1,$2,$3)", key, seed, trials).Scan(&outcome, &id)
	if err != nil {
		return 0, err
	}
	switch outcome {
	case "accepted":
		if id.Valid {
			return id.Int64, nil
		}
	case "denied":
		return 0, ErrDenied
	case "conflict":
		return 0, ErrConflict
	}
	return 0, fmt.Errorf("invalid submission decision %q", outcome)
}
