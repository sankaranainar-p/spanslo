package core

import "fmt"

// EpisodeMetrics holds the recovery metrics of one episode (R-017 to R-020). A nil pointer
// is a null in the run record: the episode is not recovered, or its turn is not derivable.
type EpisodeMetrics struct {
	TurnsToRecover  *int
	TimeToRecoverMS *int64
	CascadeSpans    int
	RecoveryTokens  int64
}

// ComputeEpisodeMetrics scores one episode. first and last are the spans of its first and
// last fault; rec is the qualifying recovery call from FindRecovery, nil when not recovered.
// Turns come from the index (chat spans), never from fault event attributes (R-005, R-007).
// When the last fault span is outside the index (such as the root span) it has no derived
// turn: turns_to_recover is null and a warning is returned; the spec defines nothing more.
// The interval of cascade_spans and recovery_tokens runs from the first fault span's start
// to the recovery call's start, both inclusive, or to the end of the run when rec is nil
// (R-019, R-020). A negative time to recover is clamped to 0 and returned as a warning for
// the caller to print (R-018).
func ComputeEpisodeMetrics(n *Names, idx *RunIndex, spans []Span, e Episode, first, last Span, rec *Call) (EpisodeMetrics, []string) {
	var m EpisodeMetrics
	var warns []string
	if rec != nil {
		if t, ok := spanTurn(idx, last); ok {
			turns := rec.T - t
			m.TurnsToRecover = &turns
		} else {
			warns = append(warns, fmt.Sprintf("episode on %s: last fault span %s is outside the index; turns_to_recover is null", e.Target, last.SpanID))
		}
		ref := last.End // same reference point as FindRecovery
		if ref.IsZero() {
			ref = last.Start
		}
		ms := rec.Span.Start.Sub(ref).Milliseconds()
		if ms < 0 {
			warns = append(warns, fmt.Sprintf("episode on %s: recovery call %s starts %d ms before the last fault span ended; time_to_recover_ms clamped to 0", e.Target, rec.Span.SpanID, -ms))
			ms = 0
		}
		m.TimeToRecoverMS = &ms
	}
	for _, s := range spans {
		if s.Start.Before(first.Start) || (rec != nil && s.Start.After(rec.Span.Start)) {
			continue
		}
		m.CascadeSpans++
		if op, _ := s.Attrs[n.Attributes.OperationName].(string); op == n.Operations.ModelCall {
			m.RecoveryTokens += tokenCount(s.Attrs[n.Attributes.InputTokens]) + tokenCount(s.Attrs[n.Attributes.OutputTokens])
		}
	}
	return m, warns
}

// spanTurn is the turn the scorer derived for a span: a chat span's own turn or a tool call's T.
// ok is false for a span outside the index, such as the root.
func spanTurn(idx *RunIndex, s Span) (int, bool) {
	for i, c := range idx.Chats {
		if c.SpanID == s.SpanID {
			return i + 1, true
		}
	}
	for _, c := range idx.Calls {
		if c.Span.SpanID == s.SpanID {
			return c.T, true
		}
	}
	return 0, false
}

// tokenCount reads a token attribute; OTLP gives int64, a double is accepted too.
func tokenCount(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case float64:
		return int64(x)
	}
	return 0
}
