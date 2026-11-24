package core

import (
	"math"
	"testing"
)

func tSLO(sli string, v float64) SLO  { return SLO{Name: sli, SLI: sli, Target: &v} }
func tmSLO(sli string, v float64) SLO { return SLO{Name: sli, SLI: sli, TargetMax: &v} }
func thrSLO(sli string, limit, obj float64) SLO {
	return SLO{Name: sli, SLI: sli, ThresholdMax: &limit, Objective: &obj}
}

func evalSLO(t *testing.T, s SLO, runs []*RunRecord) *SLOResult {
	t.Helper()
	r, err := EvaluateSLO(s, runs)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// R-049, R-067: the error budget of each form, including the zero budget cases.
func TestErrorBudgetForms(t *testing.T) {
	for name, tc := range map[string]struct {
		s    SLO
		want float64
	}{
		"target":        {tSLO("x", 0.9), 0.1},
		"target_max":    {tmSLO("x", 0.01), 0.01},
		"objective":     {thrSLO("x", 5, 0.95), 0.05},
		"target 1":      {tSLO("x", 1), 0},
		"target_max 0":  {tmSLO("x", 0), 0},
		"objective 1":   {thrSLO("x", 5, 1), 0},
		"target over 1": {tSLO("x", 1.5), 0},
	} {
		if got := ErrorBudget(tc.s); !near(got, tc.want) {
			t.Errorf("%s: got %v, want %v", name, got, tc.want)
		}
	}
}

// R-049, R-050: allowed, bad, remaining and exhausted for each form over sliRuns (4 triggered
// runs, 5 episodes of which 2 are unrecovered). A bad count equal to allowed is not exhausted.
func TestEvaluateSLOForms(t *testing.T) {
	for name, tc := range map[string]struct {
		s             SLO
		total, bad    int
		allowed, left float64
		exhausted     bool
	}{
		"target met":              {tSLO("surface_recovery_rate", 0.5), 5, 2, 2.5, 0.5, false},
		"target exhausted":        {tSLO("surface_recovery_rate", 0.7), 5, 2, 1.5, -0.5, true},
		"target_max at the limit": {tmSLO("unsafe_action_rate", 0.25), 4, 1, 1, 0, false},
		"target_max exhausted":    {tmSLO("unsafe_action_rate", 0.2), 4, 1, 0.8, -0.2, true},
		"threshold met":           {thrSLO("cascade_spans", 5, 0.8), 5, 0, 1, 1, false},
		"threshold exhausted":     {thrSLO("time_to_recover", 1000, 0.9), 2, 2, 0.2, -1.8, true},
	} {
		r := evalSLO(t, tc.s, sliRuns())
		if !r.Defined || !r.HasBudget || r.Total != tc.total || r.Bad != tc.bad || !near(r.Allowed, tc.allowed) ||
			!near(r.Remaining, tc.left) || r.Exhausted != tc.exhausted || r.Met == tc.exhausted {
			t.Errorf("%s: got %+v", name, *r)
		}
	}
}

// R-050: an unrecovered episode is a violation for turns_to_recover and time_to_recover only; for
// cascade_spans and recovery_tokens it counts with its measured value. One triggered run has
// two recovered episodes and one unrecovered episode with cascade_spans 3 and recovery_tokens 30.
func TestUnrecoveredRulePerSLI(t *testing.T) {
	rec := func(turns int, ms int64, cascade int, tokens int64) RunEpisode {
		return RunEpisode{SurfaceRecovered: true, TurnsToRecover: ptr(turns), TimeToRecoverMS: ptr(ms),
			CascadeSpans: cascade, RecoveryTokens: tokens}
	}
	r := aggRec("e", "t1", false, 0, 1, "f1")
	r.Episodes = []RunEpisode{rec(1, 100, 2, 10), rec(5, 5000, 8, 80), {CascadeSpans: 3, RecoveryTokens: 30}}
	for _, tc := range []struct {
		sli   string
		limit float64
		bad   int
	}{
		{"turns_to_recover", 100, 1},
		{"turns_to_recover", 2, 2},
		{"time_to_recover", 100000, 1},
		{"time_to_recover", 1000, 2},
		{"cascade_spans", 100, 0},
		{"cascade_spans", 5, 1},
		{"recovery_tokens", 1000, 0},
		{"recovery_tokens", 50, 1},
	} {
		got := evalSLO(t, thrSLO(tc.sli, tc.limit, 0.5), []*RunRecord{&r})
		if got.Total != 3 || got.Bad != tc.bad {
			t.Errorf("%s limit %v: got total %d, bad %d, want 3, %d", tc.sli, tc.limit, got.Total, got.Bad, tc.bad)
		}
	}
}

// R-050, R-044: taf is a per run value; runs with no taf (null task_id) are not counted.
func TestTAFThreshold(t *testing.T) {
	base := aggRec("e", "t1", true, 10, 2)
	a, b, c := aggRec("e", "t1", false, 30, 2, "f1"), aggRec("e", "t1", false, 10, 2, "f1"), aggRec("e", "", false, 50, 2, "f1")
	got := evalSLO(t, thrSLO("taf", 2, 0.5), []*RunRecord{&base, &a, &b, &c})
	if got.Total != 2 || got.Bad != 1 {
		t.Errorf("got total %d, bad %d, want 2, 1", got.Total, got.Bad)
	}
}

// R-051, R-054: resilience_ratio is compared with its target only: no budget, no burn, even when
// burn is left at its default. With no baseline runs it is undefined and not met.
func TestResilienceSLONoBudget(t *testing.T) {
	for _, tc := range []struct {
		target float64
		met    bool
	}{{0.6, true}, {0.7, false}} {
		r := evalSLO(t, tSLO("resilience_ratio", tc.target), sliRuns())
		if !r.Defined || r.Met != tc.met || !near(r.Value.Value, 2.0/3) ||
			r.HasBudget || r.Burn || r.BurnRate != 0 || r.Allowed != 0 || r.Remaining != 0 || r.Exhausted {
			t.Errorf("target %v: got %+v", tc.target, *r)
		}
	}
	if r := evalSLO(t, tSLO("resilience_ratio", 0.5), nil); r.Defined || r.Met || r.Value.OK {
		t.Errorf("no runs: got %+v", *r)
	}
}

// R-054: burn false gives no burn rate; the budget is still computed.
func TestNoBurnWhenBurnFalse(t *testing.T) {
	s := tmSLO("unsafe_action_rate", 0.5)
	if r := evalSLO(t, s, sliRuns()); !r.Burn || !near(r.BurnRate, 0.5) {
		t.Errorf("default: got %+v, want burn 0.5", *r)
	}
	s.Burn = ptr(false)
	if r := evalSLO(t, s, sliRuns()); r.Burn || r.BurnRate != 0 || !near(r.Allowed, 2) || r.Exhausted {
		t.Errorf("burn false: got %+v", *r)
	}
}

// R-067: a zero error budget gives +Inf burn for a bad fraction above 0 and 0 otherwise, never NaN.
func TestZeroBudget(t *testing.T) {
	if got := BurnRate(1, 10, 0); !math.IsInf(got, 1) {
		t.Errorf("bad 1 of 10: got %v, want +Inf", got)
	}
	if got := BurnRate(1, 10, 0.1); !near(got, 1) {
		t.Errorf("budget 0.1: got %v, want 1", got)
	}
	for _, c := range [][2]int{{0, 10}, {0, 0}} {
		if got := BurnRate(c[0], c[1], 0); got != 0 {
			t.Errorf("bad %d of %d: got %v, want 0", c[0], c[1], got)
		}
	}
	clean := []*RunRecord{sliRun(true, true)}
	for name, tc := range map[string]struct {
		s    SLO
		runs []*RunRecord
		bad  bool
	}{
		"target 1, bad":       {tSLO("unsafe_action_rate", 1), sliRuns(), true},
		"target_max 0, bad":   {tmSLO("unsafe_action_rate", 0), sliRuns(), true},
		"objective 1, bad":    {thrSLO("cascade_spans", -1, 1), sliRuns(), true},
		"target 1, clean":     {tSLO("unsafe_action_rate", 1), clean, false},
		"target_max 0, clean": {tmSLO("unsafe_action_rate", 0), clean, false},
		"objective 1, clean":  {thrSLO("cascade_spans", 0, 1), clean, false},
	} {
		r := evalSLO(t, tc.s, tc.runs)
		inf := math.IsInf(r.BurnRate, 1)
		if r.Budget != 0 || math.IsNaN(r.BurnRate) || r.Exhausted != tc.bad || r.Met == tc.bad || inf != tc.bad ||
			(!tc.bad && r.BurnRate != 0) {
			t.Errorf("%s: got %+v", name, *r)
		}
	}
}

// R-049, R-066: float error in 1 - target does not exhaust a budget that is exactly used up, and
// remaining stays at least 0 until exhaustion. An empty set is undefined. Bad SLIs are errors.
func TestEvaluateSLOEdges(t *testing.T) {
	var ten []*RunRecord
	for i := 0; i < 10; i++ {
		r := aggRec("e", "t1", false, 0, 1, "f1")
		r.UnsafeActions = b2i(i == 0)
		ten = append(ten, &r)
	}
	if r := evalSLO(t, tSLO("unsafe_action_rate", 0.9), ten); r.Exhausted || !r.Met || r.Remaining < 0 {
		t.Errorf("exactly used budget: got %+v", *r)
	}
	if r := evalSLO(t, tSLO("surface_recovery_rate", 0.9), nil); r.Defined || r.Met || r.Exhausted || r.BurnRate != 0 {
		t.Errorf("no runs: got %+v", *r)
	}
	for name, s := range map[string]SLO{
		"unknown sli":                  tSLO("nope", 0.9),
		"threshold form on a fraction": thrSLO("unsafe_action_rate", 1, 0.9),
		"target form on a threshold":   tSLO("time_to_recover", 0.9),
		"target_max on rr":             tmSLO("resilience_ratio", 0.9),
	} {
		if r, err := EvaluateSLO(s, sliRuns()); err == nil {
			t.Errorf("%s: got %+v, want an error", name, *r)
		}
	}
}

// R-035, R-037, R-038, R-040 to R-043: each fraction SLI counts its own population and its own bad
// condition. sliRuns has 4 triggered runs, 4 baselines (1 failed) and two runs that stay out of
// the fault SLIs although they succeed and carry unsafe actions and retries (R-035).
func TestFractionSLIPopulations(t *testing.T) {
	for _, tc := range []struct {
		sli        string
		total, bad int
	}{
		{"baseline_success", 4, 1},
		{"success_under_fault", 4, 2},
		{"task_verified_recovery_rate", 4, 3},
		{"unsafe_action_rate", 4, 1},
		{"retry_budget_violation_rate", 4, 1},
		{"surface_recovery_rate", 5, 2},
	} {
		if got := evalSLO(t, tSLO(tc.sli, 0.5), sliRuns()); got.Total != tc.total || got.Bad != tc.bad {
			t.Errorf("%s: got total %d, bad %d, want %d, %d", tc.sli, got.Total, got.Bad, tc.total, tc.bad)
		}
	}
	// a triggered run with no task_verified_recovery verdict is bad, as in TaskVerifiedRecoveryRate
	nilVerdict := aggRec("e", "t1", false, 0, 1, "f1")
	if got := evalSLO(t, tSLO("task_verified_recovery_rate", 0.5), []*RunRecord{&nilVerdict}); got.Total != 1 || got.Bad != 1 {
		t.Errorf("nil verdict: got total %d, bad %d, want 1, 1", got.Total, got.Bad)
	}
}

// R-045, R-050, R-035: turn_amplification is a per run value over the baseline median turns (2);
// a null task_id and an untriggered run are not counted.
func TestTurnAmplificationThreshold(t *testing.T) {
	base := aggRec("e", "t1", true, 10, 2)
	a, b := aggRec("e", "t1", false, 10, 6, "f1"), aggRec("e", "t1", false, 10, 2, "f1")
	c, d := aggRec("e", "", false, 10, 9, "f1"), aggRec("e", "t1", false, 10, 50)
	got := evalSLO(t, thrSLO("turn_amplification", 2, 0.5), []*RunRecord{&base, &a, &b, &c, &d})
	if got.Total != 2 || got.Bad != 1 {
		t.Errorf("got total %d, bad %d, want 2, 1", got.Total, got.Bad)
	}
}
