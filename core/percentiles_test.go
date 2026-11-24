package core

import "testing"

// R-046: unrecovered episodes are absent from the percentiles and counted in Unrecovered.
// The t1 baselines have median tokens 10 and median turns 2. r1 and r2 are triggered; r1 has an
// unrecovered episode with large values that must show up nowhere but the count. A run with no
// fault applied is not triggered, so its episodes are ignored.
func TestComputePercentiles(t *testing.T) {
	rec := func(turns int, ms int64, cascade int, tokens int64) RunEpisode {
		return RunEpisode{SurfaceRecovered: true, TurnsToRecover: ptr(turns), TimeToRecoverMS: ptr(ms),
			CascadeSpans: cascade, RecoveryTokens: tokens}
	}
	b := GroupBaselines([]RunRecord{aggRec("e", "t1", true, 10, 2), aggRec("e", "t1", true, 10, 2)})
	r1 := aggRec("e", "t1", false, 20, 4, "f1")
	r1.Episodes = []RunEpisode{rec(1, 100, 2, 10), {CascadeSpans: 50, RecoveryTokens: 500}}
	r2 := aggRec("e", "t1", false, 40, 6, "f1")
	r2.Episodes = []RunEpisode{rec(3, 300, 4, 30)}
	skipped := aggRec("e", "t1", false, 999, 99)
	skipped.Episodes = []RunEpisode{rec(9, 9000, 90, 900)}
	got := ComputePercentiles([]*RunRecord{&r1, &r2, &skipped}, b)
	for name, tc := range map[string]struct {
		got      Percentiles
		p50, p95 float64
	}{
		"turns_to_recover": {got.TurnsToRecover, 2, 2.9},
		"time_to_recover":  {got.TimeToRecoverMS, 200, 290},
		"cascade_spans":    {got.CascadeSpans, 3, 3.9},
		"recovery_tokens":  {got.RecoveryTokens, 20, 29},
		"taf":              {got.TAF, 3, 3.9},
		"turn_amp":         {got.TurnAmplification, 2.5, 2.95},
	} {
		if g := tc.got; !g.OK || !near(g.P50, tc.p50) || !near(g.P95, tc.p95) || g.Unrecovered != 1 {
			t.Errorf("%s: got %+v, want p50 %v, p95 %v, 1 unrecovered", name, g, tc.p50, tc.p95)
		}
	}
}

// R-046, R-066: with every episode unrecovered, or no baseline, there is no percentile, only the count.
func TestComputePercentilesUndefined(t *testing.T) {
	r := aggRec("e", "", false, 20, 4, "f1") // null task_id: taf and turn_amplification undefined
	r.Episodes = []RunEpisode{{}, {}}
	got := ComputePercentiles([]*RunRecord{&r}, nil)
	for name, p := range map[string]Percentiles{
		"turns_to_recover": got.TurnsToRecover, "time_to_recover": got.TimeToRecoverMS,
		"cascade_spans": got.CascadeSpans, "recovery_tokens": got.RecoveryTokens,
		"taf": got.TAF, "turn_amp": got.TurnAmplification,
	} {
		if p.OK || p.Unrecovered != 2 {
			t.Errorf("%s: got %+v, want undefined with 2 unrecovered", name, p)
		}
	}
	if got := ComputePercentiles(nil, nil); got != (SLIPercentiles{}) {
		t.Errorf("no runs: got %+v", got)
	}
}
