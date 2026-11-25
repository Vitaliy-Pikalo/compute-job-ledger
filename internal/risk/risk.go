// Package risk provides the ledger's deterministic demonstration workload.
package risk

import (
	"fmt"
	"math/rand"
)

const MaxTrials = 1_000_000

type Result struct {
	Seed   int64 `json:"seed"`
	Trials int   `json:"trials"`
	Losses int   `json:"losses"`
}

// Run counts synthetic loss events using a private, reproducible random stream.
func Run(seed int64, trials int) (Result, error) {
	if trials < 1 || trials > MaxTrials {
		return Result{}, fmt.Errorf("trials must be between 1 and %d", MaxTrials)
	}
	rng := rand.New(rand.NewSource(seed))
	result := Result{Seed: seed, Trials: trials}
	for i := 0; i < trials; i++ {
		if rng.Intn(100) < 5 {
			result.Losses++
		}
	}
	return result, nil
}
