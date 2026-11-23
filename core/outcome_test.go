package core

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func successEv(n *Names, score any) Event {
	er := n.Events.EvaluationResult
	attrs := map[string]any{er.Attrs.EvaluationName: er.SuccessName}
	if score != nil {
		attrs[er.Attrs.ScoreValue] = score
	}
	return Event{Name: er.Name, Attrs: attrs}
}

// finishedRoot is a root span that ends 2.5 s after it starts.
func finishedRoot(n *Names, evs ...Event) Span {
	s := mkSpan(n, "r", n.Operations.Run, 0, map[string]any{})
	s.End = s.Start.Add(2500 * time.Millisecond)
	s.Events = evs
	return s
}

// R-016, R-073: score 1.0 and 0.0 (also as integers) give success and ended_at.
func TestParseOutcome(t *testing.T) {
	n := idxNames(t)
	for name, tc := range map[string]struct {
		score any
		want  bool
	}{"pass": {1.0, true}, "fail": {0.0, false}, "int pass": {int64(1), true}, "int fail": {int64(0), false}} {
		got, err := ParseOutcome(n, []Span{mkChat(n, "c", 0), finishedRoot(n, successEv(n, tc.score))})
		if err != nil || got.Success != tc.want || got.EndedAt != "2023-11-14T22:13:22.5Z" {
			t.Errorf("%s: got %+v, %v", name, got, err)
		}
	}
}

// R-016, R-073: each case is an invalid run; the reason names the smallest span when
// several could cause it.
func TestParseOutcomeInvalid(t *testing.T) {
	n := idxNames(t)
	er := n.Events.EvaluationResult
	unfinished := finishedRoot(n, successEv(n, 1.0))
	unfinished.End = time.Time{}
	child := finishedRoot(n, successEv(n, 1.0))
	child.ParentSpanID = "p"
	chatWith := func(id string, ev Event) Span {
		c := mkChat(n, id, 0)
		c.Events = []Event{ev}
		return c
	}
	for _, tc := range []struct {
		name, want string // want: substring of the reason
		spans      []Span
	}{
		{"missing event", "no " + er.Name + " event", []Span{finishedRoot(n)}},
		{"duplicate events", "span c:", []Span{finishedRoot(n, successEv(n, 1.0)), chatWith("c", successEv(n, 1.0))}},
		{"score 0.5", "span r:", []Span{finishedRoot(n, successEv(n, 0.5))}},
		{"score missing", "span r:", []Span{finishedRoot(n, successEv(n, nil))}},
		{"missing root", "no root", []Span{chatWith("c", successEv(n, 1.0))}},
		{"only a child root", "no root", []Span{child}},
		{"unfinished root", "span r:", []Span{unfinished}},
	} {
		_, err := ParseOutcome(n, tc.spans)
		var inv *InvalidRun
		if !errors.As(err, &inv) || !strings.Contains(inv.Reason, tc.want) {
			t.Errorf("%s: got %v, want invalid run containing %q", tc.name, err, tc.want)
		}
	}
}

// R-073: a real RFC 3339 parse; 2026-02-30 and non UTC forms are invalid.
func TestParseEndedAt(t *testing.T) {
	for _, s := range []string{"2026-11-12T10:31:45Z", "2026-11-12T10:31:45.25Z"} {
		if _, err := ParseEndedAt(s); err != nil {
			t.Errorf("%s: %v", s, err)
		}
	}
	for _, s := range []string{"2026-02-30T00:00:00Z", "2026-11-12T10:31:45+00:00", "2026-11-12T10:31:45", "x"} {
		if _, err := ParseEndedAt(s); err == nil {
			t.Errorf("%s: want error", s)
		}
	}
}
