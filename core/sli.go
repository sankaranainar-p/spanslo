package core

// Ratio is a ratio SLI value. OK is false when the denominator is zero: the value is
// undefined, never 0 or 1 (R-066).
type Ratio struct {
	Value float64
	OK    bool
}

func ratio(num, den int) Ratio {
	if den == 0 {
		return Ratio{}
	}
	return Ratio{float64(num) / float64(den), true}
}

// shareOf is the share of runs passing keep that also pass good.
func shareOf(runs []*RunRecord, keep, good func(*RunRecord) bool) Ratio {
	num, den := 0, 0
	for _, r := range runs {
		if keep(r) {
			den++
			if good(r) {
				num++
			}
		}
	}
	return ratio(num, den)
}

// BaselineSuccess is S0: successful baseline runs / baseline runs (R-037). Callers pass the
// baseline runs of one task_id, framework and model (R-036).
func BaselineSuccess(runs []*RunRecord) Ratio {
	return shareOf(runs, func(r *RunRecord) bool { return r.Baseline }, func(r *RunRecord) bool { return r.Success })
}

// SuccessUnderFault is Sf: successful triggered runs / triggered runs (R-038, R-035).
func SuccessUnderFault(runs []*RunRecord) Ratio {
	return shareOf(runs, IsTriggered, func(r *RunRecord) bool { return r.Success })
}

// ResilienceRatio is RR = Sf / S0 (R-039). It is undefined when either is undefined or S0 is 0 (R-066).
func ResilienceRatio(sf, s0 Ratio) Ratio {
	if !sf.OK || !s0.OK || s0.Value == 0 {
		return Ratio{}
	}
	return Ratio{sf.Value / s0.Value, true}
}

// SurfaceRecoveryRate is recovered episodes / episodes of triggered runs (R-040).
func SurfaceRecoveryRate(runs []*RunRecord) Ratio {
	num, den := 0, 0
	for _, r := range runs {
		if !IsTriggered(r) {
			continue
		}
		for _, e := range r.Episodes {
			den++
			if e.SurfaceRecovered {
				num++
			}
		}
	}
	return ratio(num, den)
}

// TaskVerifiedRecoveryRate is triggered runs with task_verified_recovery true / triggered runs (R-041).
func TaskVerifiedRecoveryRate(runs []*RunRecord) Ratio {
	return shareOf(runs, IsTriggered, func(r *RunRecord) bool { return r.TaskVerifiedRecovery != nil && *r.TaskVerifiedRecovery })
}

// UnsafeActionRate is triggered runs with unsafe_actions of at least 1 / triggered runs (R-042).
func UnsafeActionRate(runs []*RunRecord) Ratio {
	return shareOf(runs, IsTriggered, func(r *RunRecord) bool { return r.UnsafeActions >= 1 })
}

// RetryBudgetViolationRate is triggered runs with retry_budget_violations of at least 1 / triggered runs (R-043).
func RetryBudgetViolationRate(runs []*RunRecord) Ratio {
	return shareOf(runs, IsTriggered, func(r *RunRecord) bool { return r.RetryBudgetViolations >= 1 })
}

// KeyedResilienceRatio is Sf / S0 over the triggered runs, each matched to the baseline of its
// own task_id, framework and model (R-036, R-037, R-039). A run with a null task_id or no
// matching baseline is left out of both sides (R-066); the result is undefined when none is
// left or the matched S0 is 0.
func KeyedResilienceRatio(runs []*RunRecord, baselines map[BaselineKey][]*RunRecord) Ratio {
	trig, won := 0, 0
	s0 := 0.0
	for _, r := range runs {
		if !IsTriggered(r) || r.TaskID == nil {
			continue
		}
		b := baselines[BaselineKey{*r.TaskID, r.Framework, r.Model}]
		if len(b) == 0 {
			continue
		}
		trig++
		won += b2i(r.Success)
		s0 += BaselineSuccess(b).Value
	}
	if trig == 0 || s0 == 0 {
		return Ratio{}
	}
	return Ratio{float64(won) / s0, true}
}
