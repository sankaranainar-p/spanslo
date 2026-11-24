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

// FindRecoveryFor is the recovery search of an episode; it dispatches on the fault surface
// (R-012). For a model surface fault the qualifying call is the next chat span after the
// fault turn with no error status and no later chat span with an error status. The fault
// turn is the derived turn of the span carrying the fault, a chat span or a tool call span.
// Under persistent mode no chat span qualifies (R-013), so only a declared fallback tool
// can recover, as for tools. Every other surface uses the tool search, FindRecovery.
func FindRecoveryFor(idx *RunIndex, fault Span, surface, target string, fallbacks []string, persistent bool) *Call {
	if surface != "model" || persistent {
		return FindRecovery(idx, fault, target, fallbacks, persistent)
	}
	t, ok := spanTurn(idx, fault)
	if !ok || t >= len(idx.Chats) {
		return nil
	}
	later := idx.Chats[t:]
	last := -1
	for i, c := range later {
		if c.Status.Code == StatusError {
			last = i
		}
	}
	if last+1 >= len(later) {
		return nil
	}
	return &Call{Span: later[last+1], Tool: target, T: t + last + 2}
}

// cutBefore keeps the chat and tool call spans that start before next, so next's own error
// is not a "later error". Chats stay a prefix, so turn numbers do not change.
func cutBefore(idx *RunIndex, next Span) *RunIndex {
	w := &RunIndex{}
	for _, c := range idx.Chats {
		if c.Start.Before(next.Start) {
			w.Chats = append(w.Chats, c)
		}
	}
	for _, c := range idx.Calls {
		if c.Span.SpanID != next.SpanID && c.Span.Start.Before(next.Start) {
			w.Calls = append(w.Calls, c)
		}
	}
	return w
}
