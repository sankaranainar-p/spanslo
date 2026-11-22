package core

import (
	"errors"
	"testing"
)

func faultEvent(triggered bool, turn int64) Event {
	return Event{Name: FaultEventName, Attrs: map[string]any{
		faultIDAttr: "tool-timeout-30s", faultTargetAttr: "search", FaultTurnAttr: turn,
		faultTriggeredAttr: triggered, faultSourceAttr: "bench",
	}}
}

func rootWith(n *Names, attrs map[string]any, evs ...Event) Span {
	s := mkSpan(n, "r", n.Operations.Run, 0, attrs)
	s.Events = evs
	return s
}

// R-029: a baseline run with no spanslo.mode has mode none; no faults and no flag.
func TestExtractFaultsBaselineNoMode(t *testing.T) {
	n := idxNames(t)
	f, err := ExtractFaults([]Span{rootWith(n, map[string]any{BaselineAttr: true})})
	if err != nil {
		t.Fatal(err)
	}
	if !f.Baseline || f.Mode != "none" || f.FaultNotApplied || len(f.Faults) != 0 {
		t.Errorf("got %+v", f)
	}
}

// R-027: a non baseline run with no fault events is fault_not_applied.
func TestExtractFaultsNotApplied(t *testing.T) {
	n := idxNames(t)
	f, err := ExtractFaults([]Span{rootWith(n, map[string]any{ModeAttr: "single", BaselineAttr: false})})
	if err != nil {
		t.Fatal(err)
	}
	if f.Baseline || f.Mode != "single" || !f.FaultNotApplied || len(f.Faults) != 0 {
		t.Errorf("got %+v", f)
	}
}

// R-028: untriggered faults are kept but do not count as triggered; not "not applied".
func TestExtractFaultsAllUntriggered(t *testing.T) {
	n := idxNames(t)
	f, err := ExtractFaults([]Span{rootWith(n, map[string]any{ModeAttr: "single"},
		faultEvent(false, 3), faultEvent(false, 5))})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Faults) != 2 || len(f.TriggeredFaults()) != 0 || f.FaultNotApplied {
		t.Errorf("got %+v", f)
	}
}

// R-010 input: triggered faults are extracted with their fields, in span and event order.
func TestExtractFaultsFields(t *testing.T) {
	n := idxNames(t)
	f, err := ExtractFaults([]Span{rootWith(n, map[string]any{ModeAttr: "burst"},
		faultEvent(true, 2), faultEvent(false, 4))})
	if err != nil {
		t.Fatal(err)
	}
	tr := f.TriggeredFaults()
	if f.Mode != "burst" || len(tr) != 1 || tr[0].Turn != 2 || tr[0].Target != "search" ||
		tr[0].ID != "tool-timeout-30s" || tr[0].Source != "bench" || tr[0].SpanID != "r" {
		t.Errorf("got %+v", f)
	}
}

// R-029: absent mode on a non baseline run is an invalid run, returned as an error value.
func TestExtractFaultsAbsentModeInvalid(t *testing.T) {
	n := idxNames(t)
	for name, attrs := range map[string]map[string]any{
		"baseline false":  {BaselineAttr: false},
		"baseline absent": {},
	} {
		_, err := ExtractFaults([]Span{rootWith(n, attrs, faultEvent(true, 1))})
		var inv *InvalidRun
		if !errors.As(err, &inv) {
			t.Errorf("%s: got %v, want invalid run", name, err)
		}
	}
}
