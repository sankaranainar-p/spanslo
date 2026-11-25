package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sankaranainar-p/spanslo/core"
)

const sloUsage = "usage: spanslo slo --runs <file> --slo <file>"

// loadRuns reads a JSON lines file of run records.
func loadRuns(path string) ([]*core.RunRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var runs []*core.RunRecord
	for i, l := range bytes.Split(data, []byte("\n")) {
		if len(bytes.TrimSpace(l)) == 0 {
			continue
		}
		r := &core.RunRecord{}
		if err := json.Unmarshal(l, r); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, i+1, err)
		}
		runs = append(runs, r)
	}
	return runs, nil
}

// slo implements `spanslo slo` (R-061): per SLO it prints met or not met, the budget remaining
// and each burn alert that fires. Breaches still exit 0; bad input exits 2 (R-068).
func slo(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("slo", flag.ContinueOnError)
	fs.SetOutput(stderr)
	runsPath := fs.String("runs", "", "run records file (JSON lines)")
	sloPath := fs.String("slo", "", "SLO file (YAML)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *runsPath == "" || *sloPath == "" || fs.NArg() > 0 {
		fmt.Fprintln(stderr, sloUsage)
		return 2
	}
	fail := func(err error) int {
		fmt.Fprintln(stderr, "spanslo slo:", err)
		return 2
	}
	runs, err := loadRuns(*runsPath)
	if err != nil {
		return fail(err)
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
	type win struct {
		label string
		runs  []*core.RunRecord
	}
	var budgets []win
	if tw := f.Windows.Time; tw != nil {
		w, err := core.TimeWindow(runs, now, tw.Budget)
		if err != nil {
			return fail(err)
		}
		budgets = append(budgets, win{"time " + tw.Budget.String(), w})
	}
	if cw := f.Windows.Count; cw != nil {
		w, err := core.CountWindow(runs, cw.Budget)
		if err != nil {
			return fail(err)
		}
		budgets = append(budgets, win{fmt.Sprintf("count %d", cw.Budget), w})
	}
	// alert prints the burn alert of one window pair when it fires; baselines come from all runs (R-036).
	alert := func(s core.SLO, label string, long, short []*core.RunRecord, factor float64) error {
		fires, lr, sr, err := core.CheckBurnAlertIn(s, long, short, runs, factor)
		if fires {
			fmt.Fprintf(stdout, "%s: burn alert %s factor %g: long burn %.4g, short burn %.4g\n", s.Name, label, factor, lr, sr)
		}
		return err
	}
	for _, s := range f.SLOs {
		for _, b := range budgets {
			res, err := core.EvaluateSLOIn(s, b.runs, runs)
			if err != nil {
				return fail(fmt.Errorf("%s: %w", s.Name, err))
			}
			verdict := "not met"
			if res.Met {
				verdict = "met"
			}
			switch {
			case !res.Defined:
				fmt.Fprintf(stdout, "%s [%s]: undefined, no runs or episodes\n", s.Name, b.label)
			case s.SLI == "resilience_ratio": // no error budget, compared with its target only (R-051)
				fmt.Fprintf(stdout, "%s [%s]: %s, value %.4g\n", s.Name, b.label, verdict, res.Value.Value)
			default:
				fmt.Fprintf(stdout, "%s [%s]: %s, budget remaining %v\n", s.Name, b.label, verdict, res.Remaining)
			}
		}
		if !s.BurnEnabled() { // R-054
			continue
		}
		if tw := f.Windows.Time; tw != nil {
			for _, p := range tw.Burn {
				long, err := core.TimeWindow(runs, now, p.Long)
				if err != nil {
					return fail(err)
				}
				short, err := core.TimeWindow(runs, now, p.Short)
				if err != nil {
					return fail(err)
				}
				if err := alert(s, fmt.Sprintf("time %s/%s", p.Long, p.Short), long, short, p.Factor); err != nil {
					return fail(err)
				}
			}
		}
		if cw := f.Windows.Count; cw != nil {
			for _, p := range cw.Burn {
				long, err := core.CountWindow(runs, p.Long)
				if err != nil {
					return fail(err)
				}
				short, err := core.CountWindow(runs, p.Short)
				if err != nil {
					return fail(err)
				}
				if err := alert(s, fmt.Sprintf("count %d/%d", p.Long, p.Short), long, short, p.Factor); err != nil {
					return fail(err)
				}
			}
		}
	}
	return 0
}
