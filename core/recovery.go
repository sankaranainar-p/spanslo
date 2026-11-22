package core

import "slices"

// FallbackTools returns the declared fallback tools of the task: the list in the fallback
// attribute on the run's root span. Absent or empty means no fallback (R-012).
func FallbackTools(n *Names, spans []Span) []string {
	var root *Span
	for i := range spans {
		s := &spans[i]
		if op, _ := s.Attrs[n.Attributes.OperationName].(string); op != n.Operations.Run || s.ParentSpanID != "" {
			continue
		}
		if root == nil || spanCmp(*s, *root) < 0 {
			root = s
		}
	}
	if root == nil {
		return nil
	}
	vals, _ := root.Attrs[n.Spanslo.Task.FallbackTools].([]any)
	var out []string
	for _, v := range vals {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// FindRecovery returns the qualifying recovery call of an episode, or nil when it is not
// recovered (R-012, R-013). fault is the span of the episode's last fault. Only calls to
// target or a fallback that start after that span ended are looked at, in time order. The
// qualifying call is the first success after the last error on target or any fallback. In
// persistent mode a call to the target never qualifies, only a fallback can (R-013).
// A call succeeds unless its span status is error.
func FindRecovery(idx *RunIndex, fault Span, target string, fallbacks []string, persistent bool) *Call {
	ref := fault.End
	if ref.IsZero() {
		ref = fault.Start
	}
	var rel []Call
	for _, c := range idx.Calls {
		if c.Span.SpanID == fault.SpanID || c.Span.Start.Before(ref) {
			continue
		}
		if c.Tool == target || slices.Contains(fallbacks, c.Tool) {
			rel = append(rel, c)
		}
	}
	slices.SortFunc(rel, func(a, b Call) int { return spanCmp(a.Span, b.Span) })
	last := -1
	for i, c := range rel {
		if c.Span.Status.Code == StatusError {
			last = i
		}
	}
	for _, c := range rel[last+1:] {
		if !persistent || c.Tool != target {
			return &c
		}
	}
	return nil
}

// AllRecovered is the run level rollup: the run is surface recovered only if every
// episode is (R-014). A run with no episodes has nothing unrecovered.
func AllRecovered(episodes []bool) bool {
	return !slices.Contains(episodes, false)
}
