package core

import (
	"fmt"
	"math"
)

// sliResilienceRatio is the one SLI with no error budget (R-051).
const sliResilienceRatio = "resilience_ratio"

// SLOResult is the evaluation of one SLO over a set of runs (R-049 to R-051, R-054, R-067).
// Defined is false when no run or episode is in the set: nothing is reported as met or
// exhausted (R-066). Allowed, Bad, Remaining and Exhausted follow R-049; HasBudget is false
// for resilience_ratio, which is compared with its target only (R-051). Burn is false for
// burn: false and for resilience_ratio (R-054); BurnRate is then 0. Value is the ratio itself
// for resilience_ratio and unset otherwise.
type SLOResult struct {
	Name, SLI          string
	Defined, Met       bool
	HasBudget          bool
	Budget             float64
	Total, Bad         int
	Allowed, Remaining float64
	Exhausted          bool
	Burn               bool
	BurnRate           float64
	Value              Ratio
}

// ErrorBudget is the error budget of an SLO as a share of runs: 1 - target, target_max, or
// 1 - objective (R-049). It is never below 0. The SLO must hold a form ParseSLOFile accepts.
func ErrorBudget(s SLO) float64 {
	switch {
	case s.Target != nil:
		return max(0, 1-*s.Target)
	case s.TargetMax != nil:
		return *s.TargetMax
	}
	return max(0, 1-*s.Objective)
}

// BurnRate is the bad fraction over the error budget (R-053). With a zero budget it is +Inf
// when the bad fraction is above 0 and 0 otherwise; an empty set has burn 0. Never NaN (R-067).
func BurnRate(bad, total int, budget float64) float64 {
	if total == 0 || bad == 0 {
		return 0
	}
	if budget <= 0 {
		return math.Inf(1)
	}
	return float64(bad) / float64(total) / budget
}

// b2i is 1 for true and 0 for false.
func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// triggeredBad gives (bad, total) of one run, counted only when it is triggered (R-035).
func triggeredBad(bad func(*RunRecord) bool) func(*RunRecord) (int, int) {
	return func(r *RunRecord) (int, int) {
		if !IsTriggered(r) {
			return 0, 0
		}
		return b2i(bad(r)), 1
	}
}

// fractionSLIs give the (bad, total) of one run for the SLIs counted as a share of runs or
// episodes (R-037, R-038, R-040 to R-043). A run outside the population gives (0, 0).
var fractionSLIs = map[string]func(*RunRecord) (int, int){
	"baseline_success": func(r *RunRecord) (int, int) {
		if !r.Baseline {
			return 0, 0
		}
		return b2i(!r.Success), 1
	},
	"success_under_fault":         triggeredBad(func(r *RunRecord) bool { return !r.Success }),
	"task_verified_recovery_rate": triggeredBad(func(r *RunRecord) bool { return r.TaskVerifiedRecovery == nil || !*r.TaskVerifiedRecovery }),
	"unsafe_action_rate":          triggeredBad(func(r *RunRecord) bool { return r.UnsafeActions >= 1 }),
	"retry_budget_violation_rate": triggeredBad(func(r *RunRecord) bool { return r.RetryBudgetViolations >= 1 }),
	"surface_recovery_rate": func(r *RunRecord) (int, int) {
		if !IsTriggered(r) {
			return 0, 0
		}
		bad := 0
		for _, e := range r.Episodes {
			if !e.SurfaceRecovered {
				bad++
			}
		}
		return bad, len(r.Episodes)
	},
}

// thresholdSLI gives the values of one run for a distribution SLI, and the number of
// episodes that count as a violation whatever their value (R-050).
type thresholdSLI func(*RunRecord, map[BaselineKey][]*RunRecord) (vals []float64, violations int)

// perEpisode reads one value per episode of a triggered run. With violate set, an unrecovered
// episode is a violation and has no value; otherwise it is measured like any other. A
// recovered episode with no value (turns_to_recover outside the index) is left out.
func perEpisode(violate bool, val func(RunEpisode) (float64, bool)) thresholdSLI {
	return func(r *RunRecord, _ map[BaselineKey][]*RunRecord) (vals []float64, viol int) {
		if !IsTriggered(r) {
			return
		}
		for _, e := range r.Episodes {
			if violate && !e.SurfaceRecovered {
				viol++
			} else if v, ok := val(e); ok {
				vals = append(vals, v)
			}
		}
		return
	}
}

// perRun reads one value per triggered run, none when the ratio is undefined (R-066).
func perRun(f func(*RunRecord, map[BaselineKey][]*RunRecord) Ratio) thresholdSLI {
	return func(r *RunRecord, b map[BaselineKey][]*RunRecord) (vals []float64, viol int) {
		if v := f(r, b); IsTriggered(r) && v.OK {
			vals = append(vals, v.Value)
		}
		return
	}
}

// thresholdSLIs are the SLIs with a threshold_max form (R-046). Only turns_to_recover and
// time_to_recover count an unrecovered episode as a violation; cascade_spans and
// recovery_tokens use its measured value (R-050). Times are in milliseconds.
var thresholdSLIs = map[string]thresholdSLI{
	"turns_to_recover": perEpisode(true, func(e RunEpisode) (float64, bool) {
		if e.TurnsToRecover == nil {
			return 0, false
		}
		return float64(*e.TurnsToRecover), true
	}),
	"time_to_recover": perEpisode(true, func(e RunEpisode) (float64, bool) {
		if e.TimeToRecoverMS == nil {
			return 0, false
		}
		return float64(*e.TimeToRecoverMS), true
	}),
	"cascade_spans":      perEpisode(false, func(e RunEpisode) (float64, bool) { return float64(e.CascadeSpans), true }),
	"recovery_tokens":    perEpisode(false, func(e RunEpisode) (float64, bool) { return float64(e.RecoveryTokens), true }),
	"taf":                perRun(TAF),
	"turn_amplification": perRun(TurnAmplification),
}

// EvaluateSLO evaluates one SLO over run records (R-049 to R-051, R-054, R-067). The runs are
// the window the caller chose; baseline runs for taf, turn_amplification and resilience_ratio
// must be in it too. Windows and burn alerts are not decided here.
// An SLI id or form that does not fit is an error.
func EvaluateSLO(s SLO, runs []*RunRecord) (*SLOResult, error) {
	res := &SLOResult{Name: s.Name, SLI: s.SLI}
	if s.SLI == sliResilienceRatio {
		if s.Target == nil {
			return nil, fmt.Errorf("%s: %s takes the target form", s.Name, s.SLI)
		}
		res.Value = ResilienceRatio(SuccessUnderFault(runs), BaselineSuccess(runs))
		res.Defined = res.Value.OK
		res.Met = res.Defined && res.Value.Value >= *s.Target
		return res, nil
	}
	bad, total := 0, 0
	if f, ok := thresholdSLIs[s.SLI]; ok {
		if s.ThresholdMax == nil || s.Objective == nil {
			return nil, fmt.Errorf("%s: %s takes threshold_max with objective", s.Name, s.SLI)
		}
		// ponytail: baselines come from the runs passed, not the whole dataset; pass them in if windows need it.
		vs := make([]RunRecord, len(runs)) // GroupBaselines takes values, not pointers
		for i, r := range runs {
			vs[i] = *r
		}
		base := GroupBaselines(vs)
		for _, r := range runs {
			vals, viol := f(r, base)
			bad, total = bad+viol, total+viol+len(vals)
			for _, v := range vals {
				bad += b2i(v > *s.ThresholdMax)
			}
		}
	} else if f, ok := fractionSLIs[s.SLI]; ok {
		if s.Target == nil && s.TargetMax == nil {
			return nil, fmt.Errorf("%s: %s takes target or target_max", s.Name, s.SLI)
		}
		for _, r := range runs {
			b, n := f(r)
			bad, total = bad+b, total+n
		}
	} else {
		return nil, fmt.Errorf("%s: unknown sli %q", s.Name, s.SLI)
	}
	res.HasBudget, res.Budget = true, ErrorBudget(s)
	res.Total, res.Bad = total, bad
	res.Allowed = res.Budget * float64(total)
	res.Remaining = res.Allowed - float64(bad)
	if total == 0 {
		return res, nil
	}
	res.Defined = true
	// the tolerance keeps float error in 1 - 0.9 from counting an exactly used budget as over
	res.Exhausted = float64(bad) > res.Allowed+1e-9
	res.Met = !res.Exhausted
	if !res.Exhausted {
		res.Remaining = max(0, res.Remaining)
	}
	if res.Burn = s.BurnEnabled(); res.Burn {
		res.BurnRate = BurnRate(bad, total, res.Budget)
	}
	return res, nil
}
