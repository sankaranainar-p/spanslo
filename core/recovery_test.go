package core

import (
	"slices"
	"testing"
	"time"
)

func recCall(n *Names, id, tool string, ms int, fail bool) Span {
	s := mkCall(n, id, tool, ms, nil)
	s.End = s.Start.Add(50 * time.Millisecond)
	if fail {
		s.Status.Code = StatusError
	}
	return s
}

// qualifying returns the id of the qualifying call for a fault "f" on target "search", or "".
func qualifying(t *testing.T, n *Names, fallbacks []string, persistent bool, calls ...Span) string {
	t.Helper()
	spans := append([]Span{mkChat(n, "c1", 0), mkChat(n, "c2", 5000)}, calls...)
	idx := mustIndex(t, n, spans)
	var fault Span
	for _, c := range calls {
		if c.SpanID == "f" {
			fault = c
		}
	}
	if r := FindRecovery(idx, fault, "search", fallbacks, persistent); r != nil {
		return r.Span.SpanID
	}
	return ""
}

// R-012, R-013: recovery on the target, on a fallback, a later error on a fallback, and
// persistent mode where only a fallback recovers.
func TestFindRecovery(t *testing.T) {
	n := idxNames(t)
	fb := []string{"backup"}
	f := recCall(n, "f", "search", 100, true)
	for _, tc := range []struct {
		name       string
		fallbacks  []string
		persistent bool
		calls      []Span
		want       string
	}{
		{"target", nil, false, []Span{f, recCall(n, "ok", "search", 6000, false)}, "ok"},
		{"error before success", nil, false, []Span{f, recCall(n, "e", "search", 200, true), recCall(n, "ok", "search", 6000, false)}, "ok"},
		{"fallback", fb, false, []Span{f, recCall(n, "ok", "backup", 6000, false)}, "ok"},
		{"fallback not declared", nil, false, []Span{f, recCall(n, "x", "backup", 6000, false)}, ""},
		{"later error on fallback", fb, false, []Span{f, recCall(n, "ok", "backup", 6000, false), recCall(n, "e", "backup", 7000, true)}, ""},
		{"later error on target", fb, false, []Span{f, recCall(n, "ok", "backup", 6000, false), recCall(n, "e", "search", 7000, true)}, ""},
		{"unrelated tool", fb, false, []Span{f, recCall(n, "u", "other", 6000, false)}, ""},
		{"persistent target success", fb, true, []Span{f, recCall(n, "ok", "search", 6000, false)}, ""},
		{"persistent fallback", fb, true, []Span{f, recCall(n, "t", "search", 6000, false), recCall(n, "ok", "backup", 7000, false)}, "ok"},
		{"no later call", fb, false, []Span{f}, ""},
	} {
		if got := qualifying(t, n, tc.fallbacks, tc.persistent, tc.calls...); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// R-012: fallbacks come from the root span attribute; absent means none.
func TestFallbackTools(t *testing.T) {
	n := idxNames(t)
	root := mkSpan(n, "r", n.Operations.Run, 0, map[string]any{n.Spanslo.Task.FallbackTools: []any{"a", "b"}})
	if got := FallbackTools(n, []Span{mkChat(n, "c", 0), root}); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("got %v", got)
	}
	if got := FallbackTools(n, []Span{mkSpan(n, "r", n.Operations.Run, 0, map[string]any{})}); len(got) != 0 {
		t.Errorf("absent: got %v", got)
	}
}

// R-014: one unrecovered episode makes the run unrecovered.
func TestAllRecovered(t *testing.T) {
	if !AllRecovered([]bool{true, true}) || AllRecovered([]bool{true, false}) {
		t.Error("rollup wrong")
	}
}
