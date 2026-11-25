package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sankaranainar-p/spanslo/core"
)

// burnSLO is met in its budget window, with a burn pair of 4 and 2 runs at factor 1.
const burnSLO = `slos:
  - {name: unsafe, sli: unsafe_action_rate, target_max: 0.5}
windows:
  count:
    budget: 10
    burn:
      - {long: 4, short: 2, factor: 1}
`

// runGate writes the records and, when side is not nil, the sidecar next to them, then runs
// gate on the SLO file text. It returns the exit code and stdout and stderr together.
func runGate(t *testing.T, slos string, recs []core.RunRecord, side *string) (int, string) {
	t.Helper()
	var sb bytes.Buffer
	for _, r := range recs {
		b, _ := json.Marshal(r)
		sb.Write(append(b, '\n'))
	}
	dir := t.TempDir()
	runs, sloFile := filepath.Join(dir, "runs.jsonl"), filepath.Join(dir, "slo.yaml")
	write(t, runs, sb.String())
	write(t, sloFile, slos)
	if side != nil {
		write(t, runs+".invalid.jsonl", *side)
	}
	var out, errb bytes.Buffer
	code := gate([]string{"--runs", runs, "--slo", sloFile}, &out, &errb)
	return code, out.String() + errb.String()
}

// gateRuns is two triggered runs, each with the given unsafe action count; okSLO allows 0.5.
func gateRuns(unsafe int) []core.RunRecord {
	var recs []core.RunRecord
	for i := 1; i <= 2; i++ {
		recs = append(recs, sloRec(fmt.Sprintf("r%d", i), i, func(r *core.RunRecord) { r.UnsafeActions = unsafe }))
	}
	return recs
}

// burnRuns is triggered runs in time order; bad[i] gives run i+1 one unsafe action.
func burnRuns(bad ...bool) []core.RunRecord {
	var recs []core.RunRecord
	for i, b := range bad {
		recs = append(recs, sloRec(fmt.Sprintf("r%d", i+1), i+1, func(r *core.RunRecord) {
			if b {
				r.UnsafeActions = 1
			}
		}))
	}
	return recs
}

// R-063, R-068, R-073: the sidecar is checked first; then 1 for a breach, 0 when clean, 2 for
// empty input and 2 for a run record whose ended_at does not parse.
func TestGateExitCodes(t *testing.T) {
	empty, stale := "", `{"run_id":"x","reason":"bad"}`+"\n"
	badDate := []core.RunRecord{sloRec("r1", 1, func(r *core.RunRecord) { r.EndedAt = "2026-02-30T00:00:00Z" })}
	for name, tc := range map[string]struct {
		recs []core.RunRecord
		side *string
		want int
	}{
		"clean":             {gateRuns(0), &empty, 0},
		"breach":            {gateRuns(1), &empty, 1},
		"missing sidecar":   {gateRuns(0), nil, 2},
		"non empty sidecar": {gateRuns(0), &stale, 2},
		"empty input":       {nil, &empty, 2},
		"bad ended_at":      {badDate, &empty, 2},
	} {
		if code, got := runGate(t, okSLO, tc.recs, tc.side); code != tc.want {
			t.Errorf("%s: exit %d, output %q, want %d", name, code, got, tc.want)
		}
	}
}

// R-063: a burn alert alone fails the gate, and needs both windows at or above the factor.
func TestGateBurnAlert(t *testing.T) {
	empty := ""
	// 2 of 4 bad is exactly the budget (allowed 2), so the SLO is met; the long (4) and short (2)
	// windows both burn at or above factor 1, so the alert fires.
	code, got := runGate(t, burnSLO, burnRuns(false, false, true, true), &empty)
	if code != 1 || !strings.Contains(got, "burn alert") || strings.Contains(got, "not met") {
		t.Errorf("alert only: exit %d, output %q, want 1 with a burn alert and no unmet SLO", code, got)
	}
	// same bad count, but the short window is clean: no alert, so the gate passes
	if code, got := runGate(t, burnSLO, burnRuns(true, true, false, false), &empty); code != 0 {
		t.Errorf("short window clean: exit %d, output %q, want 0", code, got)
	}
}

// R-068: a usage error exits 2.
func TestGateUsage(t *testing.T) {
	var buf bytes.Buffer
	if code := gate(nil, &buf, &buf); code != 2 {
		t.Errorf("no flags: exit %d, want 2", code)
	}
}
