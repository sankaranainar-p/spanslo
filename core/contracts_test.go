package core

import (
	"fmt"
	"testing"
)

var defaultContract = &Contract{[]string{"destructive", "outbound"}, 3, 5}

// run builds count calls to tool in the given turn (1 to 3); args differ per call when vary is set.
func run(n *Names, prefix, tool string, turn, count int, vary bool) []Span {
	var out []Span
	for i := 0; i < count; i++ {
		var args any
		if vary {
			args = fmt.Sprintf(`{"i":%d}`, i)
		}
		out = append(out, mkCall(n, fmt.Sprintf("%s%d", prefix, i), tool, (turn-1)*1000+10+i, args))
	}
	return out
}

func score(t *testing.T, n *Names, c *Contract, calls ...[]Span) ContractResult {
	t.Helper()
	spans := []Span{mkChat(n, "c1", 0), mkChat(n, "c2", 1000), mkChat(n, "c3", 2000)}
	for _, cs := range calls {
		spans = append(spans, cs...)
	}
	return ScoreContracts(n, mustIndex(t, n, spans), c)
}

// R-023: a limit of N allows N+1 calls; a different tool between calls breaks the run;
// two limits exceeded in one run count once.
func TestRetryBudgetViolations(t *testing.T) {
	n := idxNames(t)
	for _, tc := range []struct {
		name  string
		calls [][]Span
		want  int
	}{
		{"4 identical allowed", [][]Span{run(n, "a", "A", 1, 4, false)}, 0},
		{"5 identical breach", [][]Span{run(n, "a", "A", 1, 5, false)}, 1},
		{"6 same tool allowed", [][]Span{run(n, "a", "A", 1, 6, true)}, 0},
		{"7 same tool breach", [][]Span{run(n, "a", "A", 1, 7, true)}, 1},
		{"other tool breaks the run", [][]Span{run(n, "a", "A", 1, 3, false), run(n, "b", "B", 2, 1, false), run(n, "c", "A", 3, 2, false)}, 0},
		{"same tool across turns", [][]Span{run(n, "a", "A", 1, 3, false), run(n, "b", "A", 2, 2, false)}, 1},
		{"both limits, one breach", [][]Span{run(n, "a", "A", 1, 7, false)}, 1},
		{"two runs, two breaches", [][]Span{run(n, "a", "A", 1, 5, false), run(n, "b", "B", 2, 1, false), run(n, "c", "A", 3, 5, false)}, 2},
	} {
		if got := score(t, n, defaultContract, tc.calls...).RetryBudgetViolations; got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

// R-021, R-022: tagged calls are unsafe by the contract's tags; calls missing either hint
// are untagged and never unsafe.
func TestUnsafeAndUntaggedCalls(t *testing.T) {
	n := idxNames(t)
	only := &Contract{[]string{"outbound"}, 3, 5}
	none := &Contract{[]string{}, 3, 5}
	for _, tc := range []struct {
		name string
		c    *Contract
		d, o any
		unsafe int
		untagged int
	}{
		{"destructive", defaultContract, true, false, 1, 0},
		{"outbound", defaultContract, false, true, 1, 0},
		{"both false", defaultContract, false, false, 0, 0},
		{"no hints", defaultContract, nil, nil, 0, 1},
		{"one hint only", defaultContract, true, nil, 0, 1},
		{"non boolean hint", defaultContract, "true", false, 0, 1},
		{"tag not in contract", only, true, false, 0, 0},
		{"empty tags", none, true, true, 0, 0},
	} {
		s := mkCall(n, "x", "tool", 10, nil)
		if tc.d != nil {
			s.Attrs[n.Spanslo.Tool.DestructiveHint] = tc.d
		}
		if tc.o != nil {
			s.Attrs[n.Spanslo.Tool.OpenWorldHint] = tc.o
		}
		got := score(t, n, tc.c, []Span{s})
		if got.UnsafeActions != tc.unsafe || got.UntaggedToolCalls != tc.untagged {
			t.Errorf("%s: got %+v, want unsafe %d untagged %d", tc.name, got, tc.unsafe, tc.untagged)
		}
	}
}

// R-024: excess retries are never unsafe actions.
func TestRetriesNeverUnsafe(t *testing.T) {
	n := idxNames(t)
	calls := run(n, "a", "A", 1, 7, false)
	for _, s := range calls {
		s.Attrs[n.Spanslo.Tool.DestructiveHint] = false
		s.Attrs[n.Spanslo.Tool.OpenWorldHint] = false
	}
	got := score(t, n, defaultContract, calls)
	if got.RetryBudgetViolations != 1 || got.UnsafeActions != 0 || got.UntaggedToolCalls != 0 {
		t.Errorf("got %+v", got)
	}
}
