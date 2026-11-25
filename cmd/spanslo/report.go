package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sankaranainar-p/spanslo/core"
)

const reportUsage = "usage: spanslo report --runs <file> --out <file> [--slo <file>]"

// reportSLO is the result of one SLO over one budget window (R-061, R-062). Alerts are the
// burn alerts that fire for the window type of this row ("time" or "count").
type reportSLO struct {
	Name            string   `json:"name"`
	Window          string   `json:"window"`
	Defined         bool     `json:"defined"`
	Met             bool     `json:"met"`
	Value           *float64 `json:"value,omitempty"`
	BudgetRemaining *float64 `json:"budget_remaining,omitempty"`
	Alerts          []string `json:"alerts,omitempty"`
}

// ratioPtr is the ratio's value, or nil (JSON null) when it is undefined (R-066).
func ratioPtr(r core.Ratio) *float64 {
	if !r.OK {
		return nil
	}
	return &r.Value
}

// reportSLOs evaluates every SLO over each budget window, with the burn alerts that fire.
// Baselines come from all runs (R-036). Same window logic as slo.go.
// ponytail: copy of slo.go's window logic; share a helper if a third caller appears.
func reportSLOs(f *core.SLOFile, runs []*core.RunRecord, now time.Time) ([]reportSLO, error) {
	type win struct {
		kind, label string
		runs        []*core.RunRecord
	}
	var budgets []win
	if tw := f.Windows.Time; tw != nil {
		w, err := core.TimeWindow(runs, now, tw.Budget)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, win{"time", "time " + tw.Budget.String(), w})
	}
	if cw := f.Windows.Count; cw != nil {
		w, err := core.CountWindow(runs, cw.Budget)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, win{"count", fmt.Sprintf("count %d", cw.Budget), w})
	}
	out := []reportSLO{}
	for _, s := range f.SLOs {
		alerts := map[string][]string{} // by window type
		check := func(kind, label string, long, short []*core.RunRecord, factor float64) error {
			fires, _, _, err := core.CheckBurnAlertIn(s, long, short, runs, factor)
			if fires {
				alerts[kind] = append(alerts[kind], fmt.Sprintf("%s factor %g", label, factor))
			}
			return err
		}
		if tw := f.Windows.Time; tw != nil && s.BurnEnabled() {
			for _, p := range tw.Burn {
				long, err := core.TimeWindow(runs, now, p.Long)
				if err != nil {
					return nil, err
				}
				short, err := core.TimeWindow(runs, now, p.Short)
				if err != nil {
					return nil, err
				}
				if err := check("time", fmt.Sprintf("time %s/%s", p.Long, p.Short), long, short, p.Factor); err != nil {
					return nil, err
				}
			}
		}
		if cw := f.Windows.Count; cw != nil && s.BurnEnabled() {
			for _, p := range cw.Burn {
				long, err := core.CountWindow(runs, p.Long)
				if err != nil {
					return nil, err
				}
				short, err := core.CountWindow(runs, p.Short)
				if err != nil {
					return nil, err
				}
				if err := check("count", fmt.Sprintf("count %d/%d", p.Long, p.Short), long, short, p.Factor); err != nil {
					return nil, err
				}
			}
		}
		for _, b := range budgets {
			res, err := core.EvaluateSLOIn(s, b.runs, runs)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", s.Name, err)
			}
			r := reportSLO{Name: s.Name, Window: b.label, Defined: res.Defined, Met: res.Met, Alerts: alerts[b.kind]}
			if s.SLI == "resilience_ratio" { // no error budget (R-051)
				r.Value = ratioPtr(res.Value)
			} else if res.Defined {
				rem := res.Remaining
				r.BudgetRemaining = &rem
			}
			out = append(out, r)
		}
	}
	return out, nil
}

// report implements `spanslo report` (R-062): spanslo's own JSON with the SLI values, the
// percentiles with their unrecovered counts and, when --slo is given, each SLO result.
// The top level SLIs and percentiles cover all runs; the SLO rows cover their windows.
// Breached SLOs still exit 0; bad input exits 2 (R-068).
func report(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	runsPath := fs.String("runs", "", "run records file (JSON lines)")
	outPath := fs.String("out", "", "report output file (JSON)")
	sloPath := fs.String("slo", "", "SLO file (YAML), optional; adds the SLO results")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *runsPath == "" || *outPath == "" || fs.NArg() > 0 {
		fmt.Fprintln(stderr, reportUsage)
		return 2
	}
	fail := func(err error) int {
		fmt.Fprintln(stderr, "spanslo report:", err)
		return 2
	}
	runs, err := loadRuns(*runsPath)
	if err != nil {
		return fail(err)
	}
	now, err := core.WindowNow(runs) // rejects an unparseable ended_at (R-073, R-068)
	if err != nil {
		return fail(err)
	}
	vals := make([]core.RunRecord, len(runs))
	for i, r := range runs {
		vals[i] = *r
	}
	base := core.GroupBaselines(vals)
	slos := []reportSLO{}
	if *sloPath != "" {
		data, err := os.ReadFile(*sloPath)
		if err != nil {
			return fail(err)
		}
		f, err := core.ParseSLOFile(data)
		if err != nil {
			return fail(err)
		}
		if slos, err = reportSLOs(f, runs, now); err != nil {
			return fail(err)
		}
	}
	b, err := json.MarshalIndent(map[string]any{
		"slis": map[string]*float64{
			"baseline_success":            ratioPtr(core.BaselineSuccess(runs)),
			"success_under_fault":         ratioPtr(core.SuccessUnderFault(runs)),
			"resilience_ratio":            ratioPtr(core.KeyedResilienceRatio(runs, base)),
			"surface_recovery_rate":       ratioPtr(core.SurfaceRecoveryRate(runs)),
			"task_verified_recovery_rate": ratioPtr(core.TaskVerifiedRecoveryRate(runs)),
			"unsafe_action_rate":          ratioPtr(core.UnsafeActionRate(runs)),
			"retry_budget_violation_rate": ratioPtr(core.RetryBudgetViolationRate(runs)),
		},
		"percentiles": core.ComputePercentiles(runs, base),
		"slos":        slos,
	}, "", "  ")
	if err != nil {
		return fail(err)
	}
	if err := os.WriteFile(*outPath, append(b, '\n'), 0o644); err != nil {
		return fail(err)
	}
	return 0
}
