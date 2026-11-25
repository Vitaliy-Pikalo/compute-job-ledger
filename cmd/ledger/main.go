package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Vitaliy-Pikalo/compute-job-ledger/internal/risk"
)

func run(args []string, out, diagnostics io.Writer) int {
	if len(args) == 0 || args[0] != "risk" {
		fmt.Fprintln(diagnostics, "usage: ledger risk [-seed N] [-trials N]")
		return 2
	}
	flags := flag.NewFlagSet("risk", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	seed := flags.Int64("seed", 42, "random seed")
	trials := flags.Int("trials", 1000, "number of synthetic trials")
	if err := flags.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(diagnostics, "risk accepts flags only")
		return 2
	}
	result, err := risk.Run(*seed, *trials)
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 2
	}
	if err := json.NewEncoder(out).Encode(result); err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
