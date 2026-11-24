package core

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }

const pinnedSemconv = "e6658f14c9d9ea0760825f029cb82e7673f48fab"

// scoreRoot is a finished root span with the identity attributes the harness writes.
func scoreRoot(n *Names) Span {
	r := finishedRoot(n, successEv(n, 1.0))
	r.TraceID = "trace1"
	r.Attrs[ModeAttr] = "single"
	id := n.Spanslo.RunIdentity
	r.Attrs[id.ExperimentID], r.Attrs[id.Framework], r.Attrs[id.Model] = "exp1", "fw", "m"
	return r
}

// scoreFixture has two chats (15 and 27 tokens) and two calls to one tool, the second failed.
func scoreFixture(n *Names) []Span {
	return []Span{scoreRoot(n), tokChat(n, "c1", 0, 10, 5), tokChat(n, "c2", 1000, 20, 7),
		recCall(n, "k1", "search", 100, false), recCall(n, "k2", "search", 1100, true)}
}

func mustScore(t *testing.T, n *Names, spans []Span) (*RunRecord, []string) {
	t.Helper()
	r, warns, inv := ScoreRun(n, defaultContract, spans)
	if inv != nil {
		t.Fatalf("invalid run: %s", inv.Reason)
	}
	return r, warns
}

func TestScoreRunNames(t *testing.T) {
	n := idxNames(t)
	id := n.Spanslo.RunIdentity
	for _, p := range [][2]string{
		{id.RunID, "spanslo.run.id"}, {id.ExperimentID, "spanslo.experiment.id"},
		{id.TaskID, "spanslo.task.id"}, {id.Framework, "spanslo.framework"}, {id.Model, "spanslo.model"},
		{n.Spanslo.Task.TurnsExpected, "spanslo.task.turns_expected"}, {n.Semconv.Commit, pinnedSemconv},
	} {
		if p[0] != p[1] {
			t.Errorf("got %q, want %q", p[0], p[1])
		}
	}
}

// R-030, R-073 to R-079: counts, identity, version and ended_at come from the trace.
func TestScoreRunFields(t *testing.T) {
	n := idxNames(t)
	spans := scoreFixture(n)
	spans[0].Attrs[n.Spanslo.Task.TurnsExpected] = int64(3)
	spans[0].Attrs[n.Spanslo.RunIdentity.TaskID] = "task-1"
	r, warns := mustScore(t, n, spans)
	want := RunRecord{
		RunID: "trace1", ExperimentID: "exp1", TaskID: ptr("task-1"), TaskTurnsExpected: ptr(3),
		Framework: "fw", Model: "m", Mode: "single", Faults: []RunFault{}, Episodes: []RunEpisode{},
		Success: true, Turns: 2, ModelCalls: 2, ToolCalls: 2, TokensTotal: 42, UntaggedToolCalls: 2,
		FaultNotApplied: true, EndedAt: "2023-11-14T22:13:22.5Z", SemconvVersion: pinnedSemconv,
	}
	if !reflect.DeepEqual(*r, want) || len(warns) != 0 {
		t.Errorf("got %+v (warnings %v), want %+v", *r, warns, want)
	}
	spans[0].Attrs[n.Spanslo.RunIdentity.RunID] = "run-7"
	if r, _ := mustScore(t, n, spans); r.RunID != "run-7" {
		t.Errorf("run_id: got %q, want run-7", r.RunID)
	}
}

// R-031: every required key of the schema is written, null ones included; arrays are never null.
func TestScoreRunJSONKeys(t *testing.T) {
	n := idxNames(t)
	r, _ := mustScore(t, n, scoreFixture(n))
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	keys := []string{"run_id", "experiment_id", "task_id", "task_turns_expected", "framework", "model",
		"baseline", "mode", "faults", "episodes", "success", "task_verified_recovery", "turns",
		"model_calls", "tool_calls", "tokens_total", "unsafe_actions", "untagged_tool_calls",
		"retry_budget_violations", "graceful_error", "fault_not_applied", "turn_mismatches",
		"ended_at", "semconv_version"}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %s", k)
		}
	}
	if len(m) != len(keys) {
		t.Errorf("got %d keys, want %d", len(m), len(keys))
	}
	for _, k := range []string{"faults", "episodes"} {
		if _, ok := m[k].([]any); !ok {
			t.Errorf("%s: got %v, want an array", k, m[k])
		}
	}
}

// R-077: turns_expected is null when absent, not an integer or below 1; task_id is null when absent.
func TestScoreRunTurnsExpected(t *testing.T) {
	n := idxNames(t)
	for name, tc := range map[string]struct {
		v    any
		want int // -1 is null
	}{
		"absent": {nil, -1}, "five": {int64(5), 5}, "one": {int64(1), 1}, "zero": {int64(0), -1},
		"negative": {int64(-2), -1}, "string": {"3", -1}, "double": {3.0, -1},
	} {
		spans := scoreFixture(n)
		if tc.v != nil {
			spans[0].Attrs[n.Spanslo.Task.TurnsExpected] = tc.v
		}
		r, _ := mustScore(t, n, spans)
		got := -1
		if r.TaskTurnsExpected != nil {
			got = *r.TaskTurnsExpected
		}
		if got != tc.want || r.TaskID != nil {
			t.Errorf("%s: got %d, task_id %v, want %d and null", name, got, r.TaskID, tc.want)
		}
	}
}

// R-080: a missing, empty or non string experiment_id, framework or model is an invalid run
// whose reason names the attribute.
func TestScoreRunIdentityInvalid(t *testing.T) {
	n := idxNames(t)
	id := n.Spanslo.RunIdentity
	for _, key := range []string{id.ExperimentID, id.Framework, id.Model} {
		for _, v := range []any{nil, "", int64(3)} {
			spans := scoreFixture(n)
			if v == nil {
				delete(spans[0].Attrs, key)
			} else {
				spans[0].Attrs[key] = v
			}
			r, _, inv := ScoreRun(n, defaultContract, spans)
			if r != nil || inv == nil || !strings.Contains(inv.Reason, key) {
				t.Errorf("%s = %v: got %+v, %v, want invalid run naming the attribute", key, v, r, inv)
			}
		}
	}
}

// R-016, R-073: a missing, unfinished or parented root span is an invalid run with no record
// and no panic, even with turns_expected set on it.
func TestScoreRunRootInvalid(t *testing.T) {
	n := idxNames(t)
	missing := scoreFixture(n)[1:]
	unfinished := scoreFixture(n)
	unfinished[0].End = time.Time{}
	withParent := scoreFixture(n)
	withParent[0].ParentSpanID = "p"
	withParent[0].Attrs[n.Spanslo.Task.TurnsExpected] = int64(3)
	for name, spans := range map[string][]Span{"missing": missing, "unfinished": unfinished, "root with parent": withParent} {
		if r, _, inv := ScoreRun(n, defaultContract, spans); r != nil || inv == nil {
			t.Errorf("%s root: got %+v, %v, want invalid run", name, r, inv)
		}
	}
}

// R-015 to R-020 assembled: one fault on "search" at turn 1 and a recovery call in turn 2.
func TestScoreRunEpisodes(t *testing.T) {
	n := idxNames(t)
	spans := []Span{scoreRoot(n), tokChat(n, "c1", 0, 1000, 1000), tokChat(n, "c2", 1000, 10, 5),
		faultOn(recCall(n, "f", "search", 100, true), "search", 1), recCall(n, "ok", "search", 1100, false)}
	r, warns := mustScore(t, n, spans)
	wantEp := []RunEpisode{{"search", 1, 1, 1, true, ptr(1), ptr(int64(950)), 3, 15}}
	wantF := []RunFault{{"tool-timeout-30s", 1, 0, true, false}}
	if !reflect.DeepEqual(r.Episodes, wantEp) || !reflect.DeepEqual(r.Faults, wantF) {
		t.Errorf("episodes %+v, faults %+v", r.Episodes, r.Faults)
	}
	if tri(r.TaskVerifiedRecovery) != "true" || r.FaultNotApplied || r.TurnMismatches != 0 ||
		r.TokensTotal != 2015 || r.UntaggedToolCalls != 2 || len(warns) != 0 {
		t.Errorf("got %+v, warnings %v", *r, warns)
	}
}

// R-029: a baseline run with no mode has mode none and no verdict.
func TestScoreRunBaseline(t *testing.T) {
	n := idxNames(t)
	spans := scoreFixture(n)
	delete(spans[0].Attrs, ModeAttr)
	spans[0].Attrs[BaselineAttr] = true
	r, _ := mustScore(t, n, spans)
	if !r.Baseline || r.Mode != "none" || r.FaultNotApplied || r.TaskVerifiedRecovery != nil {
		t.Errorf("got %+v", *r)
	}
}

// R-011, R-007: fault turns come from the chat derived turn of the span carrying the event.
func TestScoreRunFaultTurnsDerived(t *testing.T) {
	n := idxNames(t)
	spans := []Span{scoreRoot(n), tokChat(n, "c1", 0, 1, 1), tokChat(n, "c2", 1000, 1, 1),
		faultOn(recCall(n, "f", "search", 1100, true), "search", 1), recCall(n, "ok", "search", 1200, false)}
	r, _ := mustScore(t, n, spans)
	e := r.Episodes
	if len(e) != 1 || e[0].FirstFaultTurn != 2 || e[0].LastFaultTurn != 2 ||
		len(r.Faults) != 1 || r.Faults[0].Turn != 2 || r.TurnMismatches != 1 {
		t.Errorf("got episodes %+v, faults %+v, mismatches %d", e, r.Faults, r.TurnMismatches)
	}
}

// R-011: a triggered fault on the root span makes the run invalid, naming the span.
func TestScoreRunTriggeredFaultOutsideIndex(t *testing.T) {
	n := idxNames(t)
	spans := scoreFixture(n)
	spans[0].Events = append(spans[0].Events, faultEvent(true, 1))
	r, _, inv := ScoreRun(n, defaultContract, spans)
	if r != nil || inv == nil || !strings.Contains(inv.Reason, "span r:") {
		t.Errorf("got %+v, %v, want invalid run naming span r", r, inv)
	}
}

// R-011: an untriggered fault takes its turn from spanslo.fault.turn; a bad value is invalid.
func TestScoreRunUntriggeredFaultTurn(t *testing.T) {
	n := idxNames(t)
	try := func(mod func(map[string]any)) (*RunRecord, *InvalidRun) {
		ev := faultEvent(false, 3)
		mod(ev.Attrs)
		spans := scoreFixture(n)
		spans[0].Events = append(spans[0].Events, ev)
		r, _, inv := ScoreRun(n, defaultContract, spans)
		return r, inv
	}
	r, inv := try(func(map[string]any) {})
	if inv != nil || len(r.Faults) != 1 || r.Faults[0].Turn != 3 || r.Faults[0].Triggered || len(r.Episodes) != 0 {
		t.Fatalf("got %+v, %v, want turn 3 from the attribute", r, inv)
	}
	for name, mod := range map[string]func(map[string]any){
		"missing":  func(a map[string]any) { delete(a, FaultTurnAttr) },
		"string":   func(a map[string]any) { a[FaultTurnAttr] = "2" },
		"float":    func(a map[string]any) { a[FaultTurnAttr] = 2.0 },
		"zero":     func(a map[string]any) { a[FaultTurnAttr] = int64(0) },
		"negative": func(a map[string]any) { a[FaultTurnAttr] = int64(-1) },
	} {
		r, inv := try(mod)
		if r != nil || inv == nil || !strings.Contains(inv.Reason, FaultTurnAttr) {
			t.Errorf("%s: got %+v, %v, want invalid run naming %s", name, r, inv, FaultTurnAttr)
		}
	}
}
