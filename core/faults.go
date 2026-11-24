package core

import (
	"fmt"
	"slices"
)

// Run identity and canonical fault event attribute names (R-029, R-057). They are spanslo
// names, not gen_ai names.
const (
	ModeAttr     = "spanslo.mode"
	BaselineAttr = "spanslo.baseline"

	faultIDAttr         = "spanslo.fault.id"
	faultSurfaceAttr    = "spanslo.fault.surface"
	faultTargetAttr     = "spanslo.fault.target"
	faultCallIndexAttr  = "spanslo.fault.call_index"
	faultModeAttr       = "spanslo.fault.mode"
	faultTriggeredAttr  = "spanslo.fault.triggered"
	faultVirtualDurAttr = "spanslo.fault.virtual_duration_ms"
	faultSourceAttr     = "spanslo.fault.source"
)

// FaultEvent is one canonical fault event (R-057) with the span it is attached to.
type FaultEvent struct {
	SpanID            string
	ID, Surface       string
	Target, Mode      string
	Source            string
	Turn, CallIndex   int64
	VirtualDurationMS int64
	Triggered         bool
}

// FaultInfo is the fault view of one run: every fault event (triggered or not), the run
// mode, and the baseline and not applied flags (R-027, R-028, R-029).
type FaultInfo struct {
	Faults          []FaultEvent
	Baseline        bool
	Mode            string
	FaultNotApplied bool
}

// TriggeredFaults returns the faults that fired, in order. Untriggered faults stay in
// Faults but never start an episode (R-028).
func (f *FaultInfo) TriggeredFaults() []FaultEvent {
	var out []FaultEvent
	for _, e := range f.Faults {
		if e.Triggered {
			out = append(out, e)
		}
	}
	return out
}

// ExtractFaults reads fault events, mode and baseline from a run's spans. Spans and their
// events are visited in (start, lowercase span id) order, so the result does not depend on
// input order. A missing spanslo.mode on a non baseline run returns *InvalidRun (R-029).
func ExtractFaults(spans []Span) (*FaultInfo, error) {
	sorted := slices.Clone(spans)
	slices.SortStableFunc(sorted, spanCmp)
	f := &FaultInfo{}
	var mode string
	var haveMode, haveBaseline bool
	for _, s := range sorted {
		if v, ok := s.Attrs[ModeAttr].(string); ok && !haveMode {
			mode, haveMode = v, true
		}
		if v, ok := s.Attrs[BaselineAttr].(bool); ok && !haveBaseline {
			f.Baseline, haveBaseline = v, true
		}
		for _, e := range s.Events {
			if e.Name != FaultEventName {
				continue
			}
			a := e.Attrs
			str := func(k string) string { v, _ := a[k].(string); return v }
			num := func(k string) int64 { v, _ := a[k].(int64); return v }
			trig, _ := a[faultTriggeredAttr].(bool)
			f.Faults = append(f.Faults, FaultEvent{
				SpanID: s.SpanID, ID: str(faultIDAttr), Surface: str(faultSurfaceAttr),
				Target: str(faultTargetAttr), Mode: str(faultModeAttr), Source: str(faultSourceAttr),
				Turn: num(FaultTurnAttr), CallIndex: num(faultCallIndexAttr),
				VirtualDurationMS: num(faultVirtualDurAttr), Triggered: trig,
			})
		}
	}
	switch {
	case haveMode:
		f.Mode = mode
	case f.Baseline:
		f.Mode = "none"
	default:
		return nil, &InvalidRun{fmt.Sprintf("attribute %s is absent on a non baseline run", ModeAttr)}
	}
	f.FaultNotApplied = !f.Baseline && len(f.Faults) == 0
	return f, nil
}

// DeriveFaultTurns replaces each fault's Turn with the turn the scorer derived for the span
// carrying it, never the event's own spanslo.fault.turn, which TurnMismatches only compares
// (R-011). A triggered fault on a span outside the index (such as the root) is an invalid run.
// So is an untriggered fault, which no span carries, whose spanslo.fault.turn is not an integer
// of at least 1. Faults are in (start, span id) order, so the first failure names the smallest span.
func DeriveFaultTurns(idx *RunIndex, f *FaultInfo) error {
	for i := range f.Faults {
		e := &f.Faults[i]
		if !e.Triggered {
			if e.Turn < 1 {
				return &InvalidRun{fmt.Sprintf("span %s: %s must be an integer of at least 1", e.SpanID, FaultTurnAttr)}
			}
			continue
		}
		t, ok := spanTurn(idx, Span{SpanID: e.SpanID})
		if !ok {
			return &InvalidRun{fmt.Sprintf("span %s: triggered fault is on a span outside the index", e.SpanID)}
		}
		e.Turn = int64(t)
	}
	return nil
}
