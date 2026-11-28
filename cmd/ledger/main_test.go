package main

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/Vitaliy-Pikalo/compute-job-ledger/internal/risk"
)

func TestRiskCommand(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"risk", "-seed", "17", "-trials", "250"}, &out, io.Discard); code != 0 {
		t.Fatalf("exit %d", code)
	}
	want, _ := risk.Run(17, 250)
	var got risk.Result
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got != want {
		t.Fatalf("result %+v, error %v, want %+v", got, err, want)
	}
}

func TestInvalidArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown"}, {"risk", "-trials", "0"}, {"risk", "unexpected"}} {
		if code := run(args, io.Discard, io.Discard); code != 2 {
			t.Errorf("args %v: exit %d", args, code)
		}
	}
}

func TestClaimArguments(t *testing.T) {
	t.Setenv("LEDGER_DATABASE_URL", "")
	for _, tc := range []struct {
		args []string
		want int
	}{
		{[]string{"claim", "-h"}, 0}, {[]string{"claim", "unexpected"}, 2}, {[]string{"claim"}, 2},
	} {
		if got := run(tc.args, io.Discard, io.Discard); got != tc.want {
			t.Errorf("%v: got %d, want %d", tc.args, got, tc.want)
		}
	}
}
