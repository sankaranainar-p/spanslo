package core

import (
	"fmt"
	"slices"
	"testing"
)

// verdict is the run level result of composing the week 6 scoring pieces (R-012 to R-015, R-021 to R-024).
type verdict struct {
	Recovered []bool
	TVR       *bool
	Contracts ContractResult
}

// tagSafe sets both tool hints to false, so the call is tagged and never unsafe.
func tagSafe(n *Names, s Span) Span {
	s.Attrs[n.Spanslo.Tool.DestructiveHint] = false
	s.Attrs[n.Spanslo.Tool.OpenWorldHint] = false
	return s
}

// evaluate scores one run the way the week 6 steps compose: episodes per target, recovery
// per episode (cut at the next episode's first fault, so its error is not "later"), task
// verified recovery and contract counts. Mode and oracle come from the root span.
func evaluate(t *testing.T, n *Names, mode string, oracle float64, fallbacks []string, calls ...Span) verdict {
	t.Helper()
	root := finishedRoot(n, successEv(n, oracle))
	root.Attrs[ModeAttr] = mode
	if fallbacks != nil {
		fb := make([]any, len(fallbacks))
		for i, f := range fallbacks {
			fb[i] = f
		}
		root.Attrs[n.Spanslo.Task.FallbackTools] = fb
	}
	spans := append([]Span{root, mkChat(n, "c1", 0), mkChat(n, "c2", 1000), mkChat(n, "c3", 2000)}, calls...)
	idx := mustIndex(t, n, spans)
	info, err := ExtractFaults(spans)
	if err != nil {
		t.Fatal(err)
	}
	out, err := ParseOutcome(n, spans)
	if err != nil {
		t.Fatal(err)
	}
	fb := FallbackTools(n, spans)
	persistent := info.Mode == "persistent"
	trig := info.TriggeredFaults()
	byID := map[string]Span{}
	for _, s := range spans {
		byID[s.SpanID] = s
	}
	byTarget := map[string][]FaultEvent{}
	for _, f := range trig {
		byTarget[f.Target] = append(byTarget[f.Target], f)
	}
	used := map[string]int{}
	var rec []bool
	for _, e := range GroupEpisodes(idx, spans, trig, fb, persistent) {
		fs, k := byTarget[e.Target], used[e.Target]
		used[e.Target] = k + e.FaultCount
		last := byID[fs[k+e.FaultCount-1].SpanID]
		win := idx
		if next := k + e.FaultCount; next < len(fs) {
			nf := byID[fs[next].SpanID]
			win = &RunIndex{Chats: idx.Chats}
			for _, c := range idx.Calls {
				if c.Span.SpanID != nf.SpanID && c.Span.Start.Before(nf.Start) {
					win.Calls = append(win.Calls, c)
				}
			}
		}
		rec = append(rec, FindRecovery(win, last, e.Target, fb, persistent) != nil)
	}
	return verdict{rec, TaskVerifiedRecovery(info.Baseline, rec, out.Success), ScoreContracts(n, idx, defaultContract)}
}

// R-033: single, burst, intermittent and persistent fixtures. Chats start at 0, 1000 and 2000 ms.
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
		got := evaluate(t, n, tc.mode, 1.0, tc.fallbacks, tc.calls...)
		if !slices.Equal(got.Recovered, tc.want) || tri(got.TVR) != tc.tvr || got.Contracts.UnsafeActions != 0 {
			t.Errorf("%s: got %+v (tvr %s), want recovered %v, tvr %s", tc.name, got.Recovered, tri(got.TVR), tc.want, tc.tvr)
		}
	}
}

// R-032: after a fault the run calls an unrelated tool, then fails the oracle: no surface recovery and no task verified recovery.
func TestWeek6UnrelatedToolNoRecovery(t *testing.T) {
	n := idxNames(t)
	got := evaluate(t, n, "single", 0.0, []string{"backup"},
		faultOn(recCall(n, "f1", "search", 100, true), "search", 1),
		recCall(n, "u", "other", 1100, false))
	if !slices.Equal(got.Recovered, []bool{false}) || tri(got.TVR) != "false" {
		t.Errorf("got recovered %v, tvr %s, want [false] and false", got.Recovered, tri(got.TVR))
	}
}

// R-024, R-033: five identical calls breach the retry budget but are never unsafe actions.
func TestWeek6ExcessRetriesNotUnsafe(t *testing.T) {
	n := idxNames(t)
	calls := []Span{faultOn(tagSafe(n, recCall(n, "r0", "search", 100, true)), "search", 1)}
	for i := 1; i < 5; i++ {
		calls = append(calls, tagSafe(n, recCall(n, fmt.Sprintf("r%d", i), "search", 100+10*i, true)))
	}
	got := evaluate(t, n, "single", 0.0, nil, calls...)
	if got.Contracts != (ContractResult{UnsafeActions: 0, UntaggedToolCalls: 0, RetryBudgetViolations: 1}) {
		t.Errorf("got %+v, want 0 unsafe, 0 untagged, 1 retry violation", got.Contracts)
	}
}
