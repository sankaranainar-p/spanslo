package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sankaranainar-p/spanslo/core"
)

const gateUsage = "usage: spanslo gate --runs <file> --slo <file>"

// gate implements `spanslo gate` (R-063): exit 2 when the invalid run sidecar is missing or
// non empty or the input is bad, exit 1 when an SLO is not met or a burn alert fires, else 0.
// ponytail: an SLO with no runs or episodes in its window is undefined and does not fail the gate (R-066).
func gate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	runsPath := fs.String("runs", "", "run records file (JSON lines)")
	sloPath := fs.String("slo", "", "SLO file (YAML)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *runsPath == "" || *sloPath == "" || fs.NArg() > 0 {
		fmt.Fprintln(stderr, gateUsage)
		return 2
	}
	fail := func(err error) int {
		fmt.Fprintln(stderr, "spanslo gate:", err)
		return 2
	}
	// the sidecar is checked before anything is evaluated (R-016, R-063)
	sidecar := *runsPath + ".invalid.jsonl"
	side, err := os.Stat(sidecar)
	if err != nil {
		return fail(fmt.Errorf("invalid run sidecar: %w", err))
	}
	if side.Size() > 0 {
		return fail(fmt.Errorf("%s is not empty: score found invalid runs", sidecar))
	}
	runs, err := loadRuns(*runsPath)
	if err != nil {
		return fail(err)
	}
	if len(runs) == 0 {
		return fail(fmt.Errorf("%s has no run records", *runsPath))
	}
	data, err := os.ReadFile(*sloPath)
	if err != nil {
		return fail(err)
	}
	f, err := core.ParseSLOFile(data)
	if err != nil {
		return fail(err)
	}
	now, err := core.WindowNow(runs) // rejects an unparseable ended_at (R-073, R-068)
	if err != nil {
		return fail(err)
	}
	rows, err := reportSLOs(f, runs, now)
	if err != nil {
		return fail(err)
	}
	code := 0
	for _, r := range rows {
		if r.Defined && !r.Met {
			fmt.Fprintf(stdout, "%s [%s]: not met\n", r.Name, r.Window)
			code = 1
		}
		for _, a := range r.Alerts {
			fmt.Fprintf(stdout, "%s: burn alert %s\n", r.Name, a)
			code = 1
		}
	}
	return code
}
