package core

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"
)

// endedRun is a run record with its parsed ended_at.
type endedRun struct {
	r *RunRecord
	t time.Time
}

// byEndedAt parses ended_at and orders runs by it ascending, ties by run_id (R-052).
// An unparseable ended_at is an error: slo, report and gate exit 2 on it (R-073, R-068).
func byEndedAt(runs []*RunRecord) ([]endedRun, error) {
	out := make([]endedRun, len(runs))
	for i, r := range runs {
		t, err := ParseEndedAt(r.EndedAt)
		if err != nil {
			return nil, fmt.Errorf("run %s: %w", r.RunID, err)
		}
		out[i] = endedRun{r, t}
	}
	slices.SortStableFunc(out, func(a, b endedRun) int {
		return cmp.Or(a.t.Compare(b.t), strings.Compare(a.r.RunID, b.r.RunID))
	})
	return out, nil
}

// WindowNow is "now" for offline evaluation: the latest ended_at among the runs. It is the
// zero time when there are no runs; the wall clock is never read (R-052).
func WindowNow(runs []*RunRecord) (time.Time, error) {
	s, err := byEndedAt(runs)
	if err != nil || len(s) == 0 {
		return time.Time{}, err
	}
	return s[len(s)-1].t, nil
}

// TimeWindow returns the runs whose ended_at falls in the duration d ending at now, ordered
// by ended_at then run_id (R-052).
// ponytail: the window is (now-d, now]; R-052 does not say whether the start edge is inside.
func TimeWindow(runs []*RunRecord, now time.Time, d time.Duration) ([]*RunRecord, error) {
	s, err := byEndedAt(runs)
	if err != nil {
		return nil, err
	}
	var out []*RunRecord
	for _, e := range s {
		if e.t.After(now.Add(-d)) && !e.t.After(now) {
			out = append(out, e.r)
		}
	}
	return out, nil
}

// CountWindow returns the last n runs ordered by ended_at ascending, ties by run_id (R-052).
func CountWindow(runs []*RunRecord, n int) ([]*RunRecord, error) {
	s, err := byEndedAt(runs)
	if err != nil {
		return nil, err
	}
	s = s[max(0, len(s)-max(0, n)):]
	out := make([]*RunRecord, len(s))
	for i, e := range s {
		out[i] = e.r
	}
	return out, nil
}

// CheckBurnAlert computes the burn rate of s over the long and the short window and reports
// whether the alert fires: only when both are at or above factor (R-053). An SLO with no
// burn (burn false, resilience_ratio) or an empty window has burn rate 0 and never fires.
func CheckBurnAlert(s SLO, long, short []*RunRecord, factor float64) (fires bool, longRate, shortRate float64, err error) {
	return CheckBurnAlertIn(s, long, short, long, factor)
}

// CheckBurnAlertIn is CheckBurnAlert with baselines built from dataset, the whole input, not
// from the windows (R-036).
func CheckBurnAlertIn(s SLO, long, short, dataset []*RunRecord, factor float64) (fires bool, longRate, shortRate float64, err error) {
	l, err := EvaluateSLOIn(s, long, dataset)
	if err != nil {
		return false, 0, 0, err
	}
	sh, err := EvaluateSLOIn(s, short, dataset)
	if err != nil {
		return false, 0, 0, err
	}
	return l.Burn && sh.Burn && l.BurnRate >= factor && sh.BurnRate >= factor, l.BurnRate, sh.BurnRate, nil
}
