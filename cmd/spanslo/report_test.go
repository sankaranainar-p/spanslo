package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sankaranainar-p/spanslo/core"
)

const reportSLOFile = `slos:
  - {name: unsafe, sli: unsafe_action_rate, target_max: 0.25}
  - {name: rr, sli: resilience_ratio, target: 0.4, burn: false}
  - {name: taf, sli: taf, threshold_max: 2, objective: 0.9}
windows:
  time:
    budget: 30d
    burn:
      - {long: 96h, short: 48h, factor: 1}
  count:
    budget: 10
    burn:
      - {long: 3, short: 2, factor: 1}
`

func norm(s string) string { return strings.ToLower(strings.ReplaceAll(s, "_", "")) }

// pick walks decoded JSON by key, ignoring case and underscores.
func pick(t *testing.T, v any, path ...string) any {
	t.Helper()
	for _, k := range path {
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("path %v: %v is not an object", path, v)
		}
		found := false
		for mk, mv := range m {
			if norm(mk) == norm(k) {
				v, found = mv, true
				break
			}
		}
		if !found {
			t.Fatalf("path %v: no key %q in %v", path, k, m)
		}
	}
	return v
}

// R-062, R-061, R-068: the report of a fixture matches hand checked JSON and exits 0 although an
// SLO is breached. The fixture has a baseline and two triggered runs; f1 has one recovered and
// one unrecovered episode. By hand: Sf 1/2, S0 1, RR 0.5, recovery 1/2 episodes, verified 1/2,
// unsafe 1/2 (f1), retry 1/2 (f2). unsafe breaches (allowed 0.25 x 2 = 0.5, bad 1, remaining -0.5,
// burn 2) and fires in both window types; rr has a value and no budget; taf has no baseline
// median above 0 so it is undefined. Percentiles use the one recovered episode (turns 2, cascade
// 5); the unrecovered one (cascade 7) is excluded and counted.
func TestReportFixture(t *testing.T) {
	task := "t1"
	yes, no := true, false
	two, ms := 2, int64(1000)
	base := sloRec("b1", 1, func(r *core.RunRecord) {
		r.Baseline, r.Mode, r.Faults, r.TaskID, r.Success = true, "none", []core.RunFault{}, &task, true
	})
	f1 := sloRec("f1", 2, func(r *core.RunRecord) {
		r.TaskID, r.Success, r.TaskVerifiedRecovery, r.UnsafeActions = &task, false, &no, 1
		r.Episodes = []core.RunEpisode{
			{Target: "a", SurfaceRecovered: true, TurnsToRecover: &two, TimeToRecoverMS: &ms, CascadeSpans: 5, RecoveryTokens: 40},
			{Target: "b", CascadeSpans: 7},
		}
	})
	f2 := sloRec("f2", 3, func(r *core.RunRecord) {
		r.TaskID, r.Success, r.TaskVerifiedRecovery, r.RetryBudgetViolations = &task, true, &yes, 1
	})
	dir := t.TempDir()
	runs, sloFile, out := filepath.Join(dir, "runs.jsonl"), filepath.Join(dir, "slo.yaml"), filepath.Join(dir, "report.json")
	var sb bytes.Buffer
	for _, r := range []core.RunRecord{base, f1, f2} {
		b, _ := json.Marshal(r)
		sb.Write(append(b, '\n'))
	}
	write(t, runs, sb.String())
	write(t, sloFile, reportSLOFile)
	var stdout, stderr bytes.Buffer
	if code := report([]string{"--runs", runs, "--slo", sloFile, "--out", out}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d, stderr %q, want 0 although unsafe is breached", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var got, exp map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	pct := got["percentiles"]
	delete(got, "percentiles")
	const want = `{"slis":{"baseline_success":1,"success_under_fault":0.5,"resilience_ratio":0.5,
"surface_recovery_rate":0.5,"task_verified_recovery_rate":0.5,"unsafe_action_rate":0.5,"retry_budget_violation_rate":0.5},
"slos":[
{"name":"unsafe","window":"time 720h0m0s","defined":true,"met":false,"budget_remaining":-0.5,"alerts":["time 96h0m0s/48h0m0s factor 1"]},
{"name":"unsafe","window":"count 10","defined":true,"met":false,"budget_remaining":-0.5,"alerts":["count 3/2 factor 1"]},
{"name":"rr","window":"time 720h0m0s","defined":true,"met":true,"value":0.5},
{"name":"rr","window":"count 10","defined":true,"met":true,"value":0.5},
{"name":"taf","window":"time 720h0m0s","defined":false,"met":false},
{"name":"taf","window":"count 10","defined":false,"met":false}]}`
	if err := json.Unmarshal([]byte(want), &exp); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("report:\n%s\nwant:\n%s", data, want)
	}
	for _, c := range []struct {
		path []string
		want float64
	}{
		{[]string{"turns_to_recover", "p50"}, 2}, {[]string{"turns_to_recover", "p95"}, 2},
		{[]string{"turns_to_recover", "unrecovered"}, 1},
		{[]string{"cascade_spans", "p50"}, 5}, {[]string{"cascade_spans", "p95"}, 5},
		{[]string{"cascade_spans", "unrecovered"}, 1},
	} {
		if v := pick(t, pct, c.path...); v != c.want {
			t.Errorf("percentiles %v: got %v, want %v", c.path, v, c.want)
		}
	}
}

// R-068: a usage error and an unparseable ended_at exit 2.
func TestReportExit2(t *testing.T) {
	var buf bytes.Buffer
	if code := report(nil, &buf, &buf); code != 2 {
		t.Errorf("no flags: exit %d, want 2", code)
	}
	dir := t.TempDir()
	runs := filepath.Join(dir, "runs.jsonl")
	b, _ := json.Marshal(sloRec("r1", 1, func(r *core.RunRecord) { r.EndedAt = "2026-02-30T00:00:00Z" }))
	write(t, runs, string(b)+"\n")
	if code := report([]string{"--runs", runs, "--out", filepath.Join(dir, "r.json")}, &buf, &buf); code != 2 {
		t.Errorf("bad ended_at: exit %d, want 2", code)
	}
}
