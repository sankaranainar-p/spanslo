package core

import (
	"testing"
	"testing/quick"
)

// w7ptrs returns pointers to the runs, as the SLI functions take them.
func w7ptrs(runs ...RunRecord) []*RunRecord {
	out := make([]*RunRecord, len(runs))
	for i := range runs {
		out[i] = &runs[i]
	}
	return out
}

// R-047, R-066, R-028: zero baselines and untriggered faults give undefined results, never 0 or 1.
func TestWeek7ZeroBaselinesAndUntriggered(t *testing.T) {
	untrig := aggRec("e", "t1", false, 10, 2, "!f1")
	untrig.Episodes = []RunEpisode{{}}
	notApplied := aggRec("e", "t1", false, 10, 2)
	failedBase := aggRec("e", "t1", true, 10, 2)
	trig := aggRec("e", "t1", false, 10, 2, "f1")
	trig.Success = true
	for name, runs := range map[string][]*RunRecord{
		"no runs":                  nil,
		"untriggered only":         w7ptrs(untrig, notApplied),
		"baseline and untriggered": w7ptrs(failedBase, untrig, notApplied),
	} {
		for rn, r := range map[string]Ratio{
			"sf": SuccessUnderFault(runs), "surface": SurfaceRecoveryRate(runs),
			"tvr": TaskVerifiedRecoveryRate(runs), "unsafe": UnsafeActionRate(runs),
			"retry": RetryBudgetViolationRate(runs),
		} {
			if r.OK {
				t.Errorf("%s %s: got %+v, want undefined", name, rn, r)
			}
		}
		if got := ComputePercentiles(runs, nil); got != (SLIPercentiles{}) {
			t.Errorf("%s: percentiles got %+v, want none", name, got)
		}
		if res := evalSLO(t, tmSLO("unsafe_action_rate", 0.5), runs); res.Defined || res.Met || res.Exhausted {
			t.Errorf("%s: slo got %+v, want undefined", name, *res)
		}
	}
	s0Zero := BaselineSuccess(w7ptrs(failedBase))
	for name, rr := range map[string]Ratio{
		"no baseline runs":  ResilienceRatio(SuccessUnderFault(w7ptrs(trig)), BaselineSuccess(w7ptrs(trig))),
		"no triggered runs": ResilienceRatio(SuccessUnderFault(w7ptrs(untrig)), BaselineSuccess(w7ptrs(failedBase))),
		"S0 is 0":           ResilienceRatio(SuccessUnderFault(w7ptrs(trig)), s0Zero),
	} {
		if rr.OK {
			t.Errorf("RR %s: got %+v, want undefined", name, rr)
		}
	}
	if !s0Zero.OK || s0Zero.Value != 0 {
		t.Errorf("S0 of failed baselines: got %+v, want defined 0", s0Zero)
	}
	if got := TAF(&trig, GroupBaselines(nil)); got.OK {
		t.Errorf("TAF with no baselines: got %+v, want undefined", got)
	}
}

// R-055: RR is 1.0 when the faulted runs succeed exactly as often as the baselines.
func TestWeek7PropertyRRIsOne(t *testing.T) {
	f := func(n, k uint8) bool {
		total := int(n)%20 + 1
		pass := int(k)%total + 1
		var base, faulted []RunRecord
		for i := 0; i < total; i++ {
			b := aggRec("e", "t1", true, 10, 2)
			b.Success = i < pass
			c := aggRec("e", "t1", false, 10, 2, "f1")
			c.Success = i < pass
			base, faulted = append(base, b), append(faulted, c)
		}
		rr := ResilienceRatio(SuccessUnderFault(w7ptrs(faulted...)), BaselineSuccess(w7ptrs(base...)))
		return rr.OK && near(rr.Value, 1)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

// R-055: taf (and turn_amplification) is 1.0 for a run identical to its baselines.
func TestWeek7PropertyTAFIsOne(t *testing.T) {
	f := func(tok uint16, turns, m uint8) bool {
		tk, tu := int64(tok)+1, int(turns)+1
		var base []RunRecord
		for i := 0; i <= int(m)%5; i++ {
			base = append(base, aggRec("e", "t1", true, tk, tu))
		}
		r := aggRec("e", "t1", false, tk, tu, "f1")
		b := GroupBaselines(base)
		taf, amp := TAF(&r, b), TurnAmplification(&r, b)
		return taf.OK && near(taf.Value, 1) && amp.OK && near(amp.Value, 1)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

// R-055, R-049: whenever bad is at most allowed, remaining is at least 0 and the budget is not exhausted.
func TestWeek7PropertyRemainingNeverNegative(t *testing.T) {
	f := func(total, bad, budget uint8) bool {
		n := int(total)%30 + 1
		nb := int(bad) % (n + 1)
		tm := float64(budget%101) / 100
		var runs []*RunRecord
		for i := 0; i < n; i++ {
			r := aggRec("e", "t1", false, 0, 1, "f1")
			r.UnsafeActions = b2i(i < nb)
			runs = append(runs, &r)
		}
		res, err := EvaluateSLO(tmSLO("unsafe_action_rate", tm), runs)
		if err != nil || !res.Defined {
			return false
		}
		if float64(nb) <= tm*float64(n) {
			return res.Remaining >= 0 && !res.Exhausted && res.Met
		}
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
