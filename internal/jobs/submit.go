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

func Submit(ctx context.Context, db *sql.DB, key string, seed int64, trials int) (int64, error) {
	if key == "" || trials < 1 || trials > risk.MaxTrials {
		return 0, errors.New("expected a request ID and a valid trial count")
	}
	hash := InputHash(seed, trials)
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO public.jobs(request_id,input_hash,seed,trials)
 VALUES($1,$2,$3,$4) ON CONFLICT(request_id) DO NOTHING RETURNING id`, key, hash[:], seed, trials).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		// A separate READ COMMITTED statement sees the winning concurrent insert.
		err = tx.QueryRowContext(ctx, `SELECT id FROM public.jobs WHERE request_id=$1
 AND input_hash=$2 AND seed=$3 AND trials=$4`, key, hash[:], seed, trials).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrConflict
		}
	}
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}
