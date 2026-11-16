package core

import (
	"reflect"
	"testing"
)

// R-003: spans with spanslo.run.id group by it, spans without it group by trace ID.
func TestGroupRuns(t *testing.T) {
	withID := func(trace, span, run string) Span {
		return Span{TraceID: trace, SpanID: span, Attrs: map[string]any{RunIDAttr: run}}
	}
	noID := func(trace, span string) Span {
		return Span{TraceID: trace, SpanID: span}
	}
	for _, tc := range []struct {
		name  string
		spans []Span
		want  map[string][]string // run key -> span ids in order
	}{
		{"empty", nil, map[string][]string{}},
		{"run id across traces", []Span{withID("t1", "a", "r1"), withID("t2", "b", "r1")},
			map[string][]string{"r1": {"a", "b"}}},
		{"no run id falls back to trace", []Span{noID("t1", "a"), noID("t1", "b"), noID("t2", "c")},
			map[string][]string{"t1": {"a", "b"}, "t2": {"c"}}},
		{"mixed in one trace", []Span{withID("t1", "a", "r1"), noID("t1", "b")},
			map[string][]string{"r1": {"a"}, "t1": {"b"}}},
		{"empty or non string run id is absent", []Span{
			{TraceID: "t1", SpanID: "a", Attrs: map[string]any{RunIDAttr: ""}},
			{TraceID: "t1", SpanID: "b", Attrs: map[string]any{RunIDAttr: int64(5)}}},
			map[string][]string{"t1": {"a", "b"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := map[string][]string{}
			for k, spans := range GroupRuns(tc.spans) {
				for _, s := range spans {
					got[k] = append(got[k], s.SpanID)
				}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
