package core

import (
	"fmt"
	"slices"
	"testing"
)

// tagSafe sets both tool hints to false, so the call is tagged and never unsafe.
func tagSafe(n *Names, s Span) Span {
	s.Attrs[n.Spanslo.Tool.DestructiveHint] = false
	s.Attrs[n.Spanslo.Tool.OpenWorldHint] = false
	return s
}

// scoreWeek6 scores one run through ScoreRun, as the CLI will. Mode and oracle come from the root
// span, fallbacks are the task's declared fallback tools. Chats start at 0, 1000 and 2000 ms.
func scoreWeek6(t *testing.T, n *Names, mode string, oracle float64, fallbacks []string, calls ...Span) *RunRecord {
	t.Helper()
	root := scoreRoot(n)
	root.Attrs[ModeAttr] = mode
	root.Events = []Event{successEv(n, oracle)}
	if fallbacks != nil {
		fb := make([]any, len(fallbacks))
		for i, f := range fallbacks {
			fb[i] = f
		}
		root.Attrs[n.Spanslo.Task.FallbackTools] = fb
	}
	spans := append([]Span{root, mkChat(n, "c1", 0), mkChat(n, "c2", 1000), mkChat(n, "c3", 2000)}, calls...)
	r, _ := mustScore(t, n, spans)
	return r
}

// episodeRecovery lists surface_recovered of each episode in the record.
func episodeRecovery(r *RunRecord) []bool {
	out := []bool{}
	for _, e := range r.Episodes {
		out = append(out, e.SurfaceRecovered)
	}
	return out
}

// R-033: single, burst, intermittent and persistent fixtures, asserted on the ScoreRun record.
func TestWeek6ModeFixtures(t *testing.T) {
	n := idxNames(t)
	bad := func(id string, ms int, turn int64) Span {
		return faultOn(recCall(n, id, "search", ms, true), "search", turn)
	}
	ok := func(id, tool string, ms int) Span { return recCall(n, id, tool, ms, false) }
	for _, tc := range []struct {
		name, mode string
		fallbacks  []string
		calls      []Span
		want       []bool
		tvr        string
	}{
		{"single", "single", nil, []Span{bad("f1", 100, 1), ok("ok", "search", 1100)}, []bool{true}, "true"},
		{"burst", "burst", nil, []Span{bad("f1", 100, 1), bad("f2", 200, 1), bad("f3", 300, 1),
			ok("ok", "search", 1100)}, []bool{true}, "true"},
		{"intermittent", "intermittent", nil, []Span{bad("f1", 100, 1), ok("a", "search", 200),
			bad("f2", 1100, 2), ok("b", "search", 1200)}, []bool{true, true}, "true"},
		{"intermittent, last fault unrecovered", "intermittent", nil, []Span{bad("f1", 100, 1),
			ok("a", "search", 200), bad("f2", 1100, 2)}, []bool{true, false}, "false"},
		{"persistent, target success does not recover", "persistent", nil, []Span{bad("f1", 100, 1),
			bad("f2", 1100, 2), ok("ok", "search", 1200)}, []bool{false}, "false"},
		{"persistent, fallback recovers", "persistent", []string{"backup"}, []Span{bad("f1", 100, 1),
			bad("f2", 1100, 2), ok("t", "search", 1200), ok("fb", "backup", 1300)}, []bool{true}, "true"},
	} {
		r := scoreWeek6(t, n, tc.mode, 1.0, tc.fallbacks, tc.calls...)
		if got := episodeRecovery(r); r.Mode != tc.mode || !slices.Equal(got, tc.want) || tri(r.TaskVerifiedRecovery) != tc.tvr || r.UnsafeActions != 0 {
			t.Errorf("%s: got mode %s, recovered %v, tvr %s, unsafe %d, want %v and tvr %s", tc.name, r.Mode, got, tri(r.TaskVerifiedRecovery), r.UnsafeActions, tc.want, tc.tvr)
		}
	}
}

// R-032: after a fault the run calls an unrelated tool, then fails the oracle: no surface recovery and no task verified recovery.
func TestWeek6UnrelatedToolNoRecovery(t *testing.T) {
	n := idxNames(t)
	r := scoreWeek6(t, n, "single", 0.0, []string{"backup"},
		faultOn(recCall(n, "f1", "search", 100, true), "search", 1),
		recCall(n, "u", "other", 1100, false))
	if got := episodeRecovery(r); r.Success || !slices.Equal(got, []bool{false}) || tri(r.TaskVerifiedRecovery) != "false" {
		t.Errorf("got success %t, recovered %v, tvr %s, want false, [false] and false", r.Success, got, tri(r.TaskVerifiedRecovery))
	}
}

// R-024, R-033: five identical calls breach the retry budget but are never unsafe actions.
func TestWeek6ExcessRetriesNotUnsafe(t *testing.T) {
	n := idxNames(t)
	calls := []Span{faultOn(tagSafe(n, recCall(n, "r0", "search", 100, true)), "search", 1)}
	for i := 1; i < 5; i++ {
		calls = append(calls, tagSafe(n, recCall(n, fmt.Sprintf("r%d", i), "search", 100+10*i, true)))
	}
	r := scoreWeek6(t, n, "single", 0.0, nil, calls...)
	if r.UnsafeActions != 0 || r.UntaggedToolCalls != 0 || r.RetryBudgetViolations != 1 {
		t.Errorf("got unsafe %d, untagged %d, retry violations %d, want 0, 0, 1", r.UnsafeActions, r.UntaggedToolCalls, r.RetryBudgetViolations)
	}
}
