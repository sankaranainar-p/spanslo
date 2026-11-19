package core

import "testing"

func faultEv(turn any) Event {
	a := map[string]any{}
	if turn != nil {
		a[FaultTurnAttr] = turn
	}
	return Event{Name: FaultEventName, Attrs: a}
}

func turnSpans(n *Names) []Span {
	return []Span{mkChat(n, "c1", 0), mkChat(n, "c2", 1000), mkCall(n, "k", "t", 1500, nil),
		mkSpan(n, "r", n.Operations.Run, 0, map[string]any{})}
}

func countMismatches(t *testing.T, n *Names, spans []Span) int {
	t.Helper()
	return TurnMismatches(mustIndex(t, n, spans), spans)
}

// R-007 (a) to (e): chat c2 is turn 2, call k is in turn 2, root span r is outside the index.
func TestTurnMismatches(t *testing.T) {
	n := idxNames(t)
	for _, tc := range []struct {
		name, on string
		ev       Event
		want     int
	}{
		{"chat match", "c2", faultEv(int64(2)), 0},
		{"chat differs", "c2", faultEv(int64(1)), 1},
		{"call match", "k", faultEv(int64(2)), 0},
		{"call differs", "k", faultEv(int64(1)), 1},
		{"root span not counted", "r", faultEv(int64(9)), 0},
		{"missing turn", "c1", faultEv(nil), 1},
		{"non integer turn", "c1", faultEv("1"), 1},
		{"float turn", "c1", faultEv(1.0), 1},
	} {
		spans := turnSpans(n)
		for i := range spans {
			if spans[i].SpanID == tc.on {
				spans[i].Events = []Event{tc.ev}
			}
		}
		if got := countMismatches(t, n, spans); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

// R-007: one matching and one mismatching fault event give turn_mismatches 1.
func TestTurnMismatchesMixed(t *testing.T) {
	n := idxNames(t)
	spans := turnSpans(n)
	spans[0].Events = []Event{faultEv(int64(1))}
	spans[2].Events = []Event{faultEv(int64(1))}
	if got := countMismatches(t, n, spans); got != 1 {
		t.Errorf("got %d, want 1", got)
	}
}
