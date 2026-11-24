package core

import (
	"math"
	"testing"
)

// sliRun is a triggered run with one episode per entry of recovered.
func sliRun(success bool, recovered ...bool) *RunRecord {
	r := aggRec("e", "t", false, 0, 1, "f1")
	r.Success = success
	for _, ok := range recovered {
		r.Episodes = append(r.Episodes, RunEpisode{SurfaceRecovered: ok})
	}
	r.TaskVerifiedRecovery = TaskVerifiedRecovery(false, recovered, success)
	return &r
}

func sliBaseline(success bool) *RunRecord {
	r := aggRec("e", "t", true, 0, 1)
	r.Success = success
	return &r
}

// sliRuns: four triggered runs (2 succeed, 3 of 5 episodes recovered, 1 verified recovery, one
// with unsafe actions, one with retry violations), four baselines (3 succeed), and two runs that
// must not count anywhere: fault_not_applied and all untriggered.
func sliRuns() []*RunRecord {
	b := sliRun(false, true, false)
	b.UnsafeActions = 3
	c := sliRun(true, false)
	c.RetryBudgetViolations = 2
	skipped := []RunRecord{aggRec("e", "t", false, 0, 1), aggRec("e", "t", false, 0, 1, "!f1")}
	for i := range skipped {
		skipped[i].Success, skipped[i].UnsafeActions, skipped[i].RetryBudgetViolations = true, 1, 1
	}
	return []*RunRecord{sliRun(true, true), b, c, sliRun(false, true),
		sliBaseline(true), sliBaseline(true), sliBaseline(true), sliBaseline(false),
		&skipped[0], &skipped[1]}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// R-037 to R-043: a normal case for every SLI; R-066: no runs means undefined, not 0 or 1.
func TestRatioSLIs(t *testing.T) {
	for name, tc := range map[string]struct {
		f    func([]*RunRecord) Ratio
		want float64
	}{
		"S0": {BaselineSuccess, 0.75}, "Sf": {SuccessUnderFault, 0.5},
		"surface recovery": {SurfaceRecoveryRate, 0.6}, "task verified recovery": {TaskVerifiedRecoveryRate, 0.25},
		"unsafe": {UnsafeActionRate, 0.25}, "retry": {RetryBudgetViolationRate, 0.25},
	} {
		if got := tc.f(sliRuns()); !got.OK || !near(got.Value, tc.want) {
			t.Errorf("%s: got %+v, want %v", name, got, tc.want)
		}
		if got := tc.f(nil); got != (Ratio{}) {
			t.Errorf("%s with no runs: got %+v, want undefined", name, got)
		}
	}
}

// R-066: baselines alone leave every fault SLI undefined; triggered runs alone leave S0 undefined.
func TestRatioSLIsOtherPopulation(t *testing.T) {
	var base, trig []*RunRecord
	for _, r := range sliRuns() {
		if r.Baseline {
			base = append(base, r)
		} else {
			trig = append(trig, r)
		}
	}
	if got := BaselineSuccess(trig); got.OK {
		t.Errorf("S0 with no baseline runs: got %+v", got)
	}
	for _, f := range []func([]*RunRecord) Ratio{SuccessUnderFault, SurfaceRecoveryRate,
		TaskVerifiedRecoveryRate, UnsafeActionRate, RetryBudgetViolationRate} {
		if got := f(base); got.OK {
			t.Errorf("no triggered runs: got %+v", got)
		}
	}
}

// R-039, R-066: RR is Sf / S0; undefined with no baseline runs, no triggered runs or S0 equal to 0.
func TestResilienceRatio(t *testing.T) {
	rs := sliRuns()
	if rr := ResilienceRatio(SuccessUnderFault(rs), BaselineSuccess(rs)); !rr.OK || !near(rr.Value, 2.0/3) {
		t.Errorf("normal: got %+v, want 2/3", rr)
	}
	zero := []*RunRecord{sliBaseline(false), sliRun(true, true)}
	if s0 := BaselineSuccess(zero); !s0.OK || s0.Value != 0 {
		t.Fatalf("S0: got %+v, want defined 0", s0)
	}
	for name, rr := range map[string]Ratio{
		"S0 equal to 0":     ResilienceRatio(SuccessUnderFault(zero), BaselineSuccess(zero)),
		"no baseline runs":  ResilienceRatio(SuccessUnderFault(rs), BaselineSuccess(nil)),
		"no triggered runs": ResilienceRatio(SuccessUnderFault(nil), BaselineSuccess(rs)),
	} {
		if rr.OK {
			t.Errorf("%s: got %+v, want undefined", name, rr)
		}
	}
}
