package core

// GroupRuns groups spans into runs (R-003). The key is the span's spanslo.run.id when it
// is a non empty string, otherwise the span's trace ID. Spans keep input order within a run.
// The attribute name is read from Names, not hardcoded (R-078).
// ponytail: a run id equal to another trace's ID would merge them; harness ids are not trace ids.
func GroupRuns(n *Names, spans []Span) map[string][]Span {
	attr := n.Spanslo.RunIdentity.RunID
	runs := make(map[string][]Span)
	for _, s := range spans {
		key := s.TraceID
		if id, ok := s.Attrs[attr].(string); ok && id != "" {
			key = id
		}
		runs[key] = append(runs[key], s)
	}
	return runs
}
