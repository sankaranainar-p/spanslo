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

const okSLO = `slos:
  - {name: unsafe, sli: unsafe_action_rate, target_max: 0.5}
windows:
  count: {budget: 10}
`

// sloRec builds a triggered run record that ended on the given day of November 2026.
func sloRec(id string, day int, mut func(*core.RunRecord)) core.RunRecord {
	r := core.RunRecord{RunID: id, ExperimentID: "e", Framework: "fw", Model: "m", Mode: "single",
		Faults: []core.RunFault{{ID: "f", Triggered: true}}, Episodes: []core.RunEpisode{},
		EndedAt: fmt.Sprintf("2026-11-%02dT00:00:00Z", day)}
	if mut != nil {
		mut(&r)
	}
	return r
}

// runSLO writes the records, then extra text, and the SLO file, and runs slo. It returns the
// exit code and stdout and stderr together.
func runSLO(t *testing.T, slos string, recs []core.RunRecord, extra string) (int, string) {
	t.Helper()
	var sb strings.Builder
	for _, r := range recs {
		b, _ := json.Marshal(r)
		sb.Write(b)
		sb.WriteByte('\n')
	}
	sb.WriteString(extra)
	dir := t.TempDir()
	runs, sloFile := filepath.Join(dir, "runs.jsonl"), filepath.Join(dir, "slo.yaml")
	write(t, runs, sb.String())
	write(t, sloFile, slos)
	var out, errb bytes.Buffer
	code := slo([]string{"--runs", runs, "--slo", sloFile}, &out, &errb)
	return code, out.String() + errb.String()
}

// wantAll fails for each of the substrings the output lacks.
func wantAll(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("output lacks %q:\n%s", w, got)
		}
	}
}

// R-061, R-068: a met SLO, a breached SLO and a firing burn alert are printed, and slo exits 0.
// The burn false SLO would breach and alert too, but prints no alert (R-054).
func TestSLOPrintsVerdicts(t *testing.T) {
	var recs []core.RunRecord
	for i := 1; i <= 4; i++ {
		recs = append(recs, sloRec(fmt.Sprintf("r%d", i), i, func(r *core.RunRecord) {
			if i == 1 {
				r.UnsafeActions = 1
			}
			if i == 4 {
				r.RetryBudgetViolations = 1
			}
		}))
	}
	const slos = `slos:
  - {name: unsafe, sli: unsafe_action_rate, target_max: 0.5}
  - {name: retry, sli: retry_budget_violation_rate, target_max: 0.1}
  - {name: quiet, sli: retry_budget_violation_rate, target_max: 0.1, burn: false}
windows:
  count:
    budget: 10
    burn:
      - {long: 4, short: 2, factor: 2}
`
	code, got := runSLO(t, slos, recs, "")
	if code != 0 {
		t.Fatalf("exit %d, output %q, want 0", code, got)
	}
	wantAll(t, got, "unsafe [count 10]: met, budget remaining", "retry [count 10]: not met, budget remaining",
		"quiet [count 10]: not met", "retry: burn alert")
	if strings.Count(got, "burn alert") != 1 {
		t.Errorf("want exactly one burn alert:\n%s", got)
	}
}

// R-061: a time window prints its budget line and a time burn alert.
func TestSLOTimeWindow(t *testing.T) {
	var recs []core.RunRecord
	for i := 1; i <= 4; i++ {
		recs = append(recs, sloRec(fmt.Sprintf("r%d", i), i, func(r *core.RunRecord) {
			if i >= 3 {
				r.UnsafeActions = 1
			}
		}))
	}
	const slos = `slos:
  - {name: unsafe, sli: unsafe_action_rate, target_max: 0.1}
windows:
  time:
    budget: 30d
    burn:
      - {long: 96h, short: 48h, factor: 2}
`
	code, got := runSLO(t, slos, recs, "")
	if code != 0 {
		t.Fatalf("exit %d, output %q, want 0", code, got)
	}
	wantAll(t, got, "unsafe [time 720h0m0s]: not met, budget remaining", "unsafe: burn alert time 96h0m0s/48h0m0s")
}

// R-051, R-054: resilience_ratio prints its value and no budget or alert; with no triggered
// runs an SLO is undefined.
func TestSLOResilienceAndUndefined(t *testing.T) {
	task := "t1"
	base := sloRec("b1", 1, func(r *core.RunRecord) {
		r.Baseline, r.Mode, r.Faults, r.TaskID, r.Success = true, "none", []core.RunFault{}, &task, true
	})
	faulted := sloRec("f1", 2, func(r *core.RunRecord) { r.TaskID, r.Success = &task, true })
	const rr = `slos:
  - {name: rr, sli: resilience_ratio, target: 0.8, burn: false}
windows:
  count:
    budget: 10
    burn:
      - {long: 2, short: 1, factor: 0.5}
`
	code, got := runSLO(t, rr, []core.RunRecord{base, faulted}, "")
	wantAll(t, got, "rr [count 10]: met, value 1")
	if code != 0 || strings.Contains(got, "burn alert") || strings.Contains(got, "budget remaining") {
		t.Errorf("exit %d, output %q, want 0 and no budget or alert", code, got)
	}
	code, got = runSLO(t, okSLO, []core.RunRecord{base}, "")
	wantAll(t, got, "unsafe [count 10]: undefined")
	if code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
}

// R-068, R-073: bad input to the whole command exits 2: an unparseable ended_at, a malformed
// runs line, an unparseable SLO file.
func TestSLOBadInput(t *testing.T) {
	good := sloRec("r1", 1, nil)
	badDate := sloRec("r2", 2, func(r *core.RunRecord) { r.EndedAt = "2026-02-30T00:00:00Z" })
	for name, tc := range map[string]struct {
		slos  string
		recs  []core.RunRecord
		extra string
	}{
		"bad ended_at": {okSLO, []core.RunRecord{good, badDate}, ""},
		"malformed line": {okSLO, []core.RunRecord{good}, "{not json\n"},
		"bad slo file": {"slos: [", []core.RunRecord{good}, ""},
	} {
		if code, got := runSLO(t, tc.slos, tc.recs, tc.extra); code != 2 {
			t.Errorf("%s: exit %d, output %q, want 2", name, code, got)
		}
	}
}

// R-068: a missing runs file and a usage error exit 2.
func TestSLOExit2(t *testing.T) {
	var out, errb bytes.Buffer
	if code := slo([]string{"--runs", "/nonexistent", "--slo", "/nonexistent"}, &out, &errb); code != 2 {
		t.Errorf("missing file: exit %d, want 2", code)
	}
	if code := slo(nil, &out, &errb); code != 2 {
		t.Errorf("no flags: exit %d, want 2", code)
	}
}
