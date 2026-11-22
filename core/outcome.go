package core

import (
	"fmt"
	"strings"
	"time"
)

// Outcome is what a run's task success event and root span give: success and ended_at
// (R-016, R-073).
type Outcome struct {
	Success bool
	EndedAt string // RFC 3339 UTC with a trailing Z
}

// ParseEndedAt is the real RFC 3339 parse of ended_at: UTC with a trailing Z, and a date
// that exists (2026-02-30 fails). The schema pattern alone lets such dates through (R-073).
func ParseEndedAt(s string) (time.Time, error) {
	if !strings.HasSuffix(s, "Z") {
		return time.Time{}, fmt.Errorf("ended_at %q: not UTC with a trailing Z", s)
	}
	return time.Parse(time.RFC3339, s)
}

// ParseOutcome reads success and ended_at from one run's spans. Anything wrong with this
// run returns *InvalidRun; when several spans could cause it, the reason names the span
// with the smallest (start time, lowercase span id). The root is the run span with no
// parent; a run span that has a parent does not count, so a run holding only one is invalid
// (R-016, R-073).
func ParseOutcome(n *Names, spans []Span) (*Outcome, error) {
	var root *Span
	for i := range spans {
		s := spans[i]
		if op, _ := s.Attrs[n.Attributes.OperationName].(string); op != n.Operations.Run || s.ParentSpanID != "" {
			continue
		}
		if root == nil || spanCmp(s, *root) < 0 {
			root = &s
		}
	}
	if root == nil {
		return nil, &InvalidRun{"no root " + n.Operations.Run + " span"}
	}
	if root.End.IsZero() {
		return nil, &InvalidRun{fmt.Sprintf("span %s: root span is unfinished", root.SpanID)}
	}
	ended := root.End.UTC().Format(time.RFC3339Nano)
	if _, err := ParseEndedAt(ended); err != nil {
		return nil, &InvalidRun{fmt.Sprintf("span %s: %v", root.SpanID, err)}
	}

	er := n.Events.EvaluationResult
	var holder *Span
	var val any
	count := 0
	for i := range spans {
		s := spans[i]
		for _, e := range s.Events {
			if e.Name != er.Name || e.Attrs[er.Attrs.EvaluationName] != er.SuccessName {
				continue
			}
			count++
			if holder == nil || spanCmp(s, *holder) < 0 {
				holder, val = &s, e.Attrs[er.Attrs.ScoreValue]
			}
		}
	}
	if count == 0 {
		return nil, &InvalidRun{fmt.Sprintf("no %s event named %s", er.Name, er.SuccessName)}
	}
	if count > 1 {
		return nil, &InvalidRun{fmt.Sprintf("span %s: %d %s events named %s, want one", holder.SpanID, count, er.Name, er.SuccessName)}
	}
	var f float64
	switch v := val.(type) {
	case float64:
		f = v
	case int64:
		f = float64(v)
	default:
		return nil, &InvalidRun{fmt.Sprintf("span %s: score value is missing or not a number", holder.SpanID)}
	}
	switch f {
	case er.SuccessValuePass:
		return &Outcome{true, ended}, nil
	case er.SuccessValueFail:
		return &Outcome{false, ended}, nil
	}
	return nil, &InvalidRun{fmt.Sprintf("span %s: score value %v is not %v or %v", holder.SpanID, f, er.SuccessValuePass, er.SuccessValueFail)}
}
