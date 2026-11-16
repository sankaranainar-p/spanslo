package core

// RunIDAttr is the span attribute that carries the harness run id (R-003).
const RunIDAttr = "spanslo.run.id"

// GroupRuns groups spans into runs (R-003). The key is the span's spanslo.run.id when it
// is a non empty string, otherwise the span's trace ID. Spans keep input order within a run.
// ponytail: a run id equal to another trace's ID would merge them; harness ids are not trace ids.
func GroupRuns(spans []Span) map[string][]Span {
	runs := make(map[string][]Span)
	for _, s := range spans {
		key := s.TraceID
		if id, ok := s.Attrs[RunIDAttr].(string); ok && id != "" {
			key = id
		}
		runs[key] = append(runs[key], s)
	}
	return runs
}
