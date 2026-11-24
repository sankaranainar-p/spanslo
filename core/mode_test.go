package core

import (
	"os"
	"strings"
	"testing"
)

// R-078: GroupRuns reads the run id attribute name from Names, and core/runs.go holds no
// literal of it.
func TestGroupRunsUsesNames(t *testing.T) {
	n := idxNames(t)
	src, err := os.ReadFile("runs.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), `"`+n.Spanslo.RunIdentity.RunID+`"`) {
		t.Error("core/runs.go hardcodes the run id attribute name")
	}
	custom := *n
	custom.Spanslo.RunIdentity.RunID = "custom.run"
	spans := []Span{
		{TraceID: "t1", SpanID: "a", Attrs: map[string]any{"custom.run": "r1"}},
		{TraceID: "t2", SpanID: "b", Attrs: map[string]any{"custom.run": "r1"}},
		{TraceID: "t3", SpanID: "c", Attrs: map[string]any{n.Spanslo.RunIdentity.RunID: "r9"}},
	}
	runs := GroupRuns(&custom, spans)
	if len(runs["r1"]) != 2 || len(runs["t3"]) != 1 || len(runs) != 2 {
		t.Errorf("got %v", runs)
	}
}

// R-029: a present spanslo.mode must be one of the five values, and none exactly for baseline
// runs; every other case is an invalid run whose reason names spanslo.mode.
func TestScoreRunModeInvalid(t *testing.T) {
	n := idxNames(t)
	for name, tc := range map[string]struct {
		mode     any
		baseline bool
		fault    bool
	}{
		"unknown mode":          {"weird", false, false},
		"non string mode":       {int64(3), false, false},
		"none on faulted run":   {"none", false, true},
		"none on unfaulted run": {"none", false, false},
		"single on baseline":    {"single", true, false},
	} {
		spans := scoreFixture(n)
		spans[0].Attrs[ModeAttr] = tc.mode
		if tc.baseline {
			spans[0].Attrs[BaselineAttr] = true
		}
		if tc.fault {
			spans[0].Events = append(spans[0].Events, faultEvent(false, 1))
		}
		r, _, inv := ScoreRun(n, defaultContract, spans)
		if r != nil || inv == nil || !strings.Contains(inv.Reason, ModeAttr) {
			t.Errorf("%s: got %+v, %v, want invalid run naming %s", name, r, inv, ModeAttr)
		}
	}
	spans := scoreFixture(n)
	spans[0].Attrs[ModeAttr] = "none"
	spans[0].Attrs[BaselineAttr] = true
	if _, _, inv := ScoreRun(n, defaultContract, spans); inv != nil {
		t.Errorf("none on a baseline run: got invalid run %s", inv.Reason)
	}
}
