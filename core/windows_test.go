package core

import (
	"testing"
	"time"
)

func endAt(id, ended string, unsafe int) *RunRecord {
	r := aggRec("e", "t1", false, 0, 1, "f1")
	r.RunID, r.EndedAt, r.UnsafeActions = id, ended, unsafe
	return &r
}

func runIDs(runs []*RunRecord) string {
	s := ""
	for _, r := range runs {
		s += r.RunID + " "
	}
	return s
}

// R-052: "now" is the latest ended_at, compared as times (fractional seconds), never the clock.
func TestWindowNow(t *testing.T) {
	runs := []*RunRecord{endAt("a", "2026-11-12T10:00:00.5Z", 0), endAt("b", "2026-11-12T10:00:00Z", 0)}
	now, err := WindowNow(runs)
	if err != nil || !now.Equal(time.Date(2026, 11, 12, 10, 0, 0, 500000000, time.UTC)) {
		t.Errorf("got %v, %v", now, err)
	}
	if now, err := WindowNow(nil); err != nil || !now.IsZero() {
		t.Errorf("no runs: got %v, %v", now, err)
	}
	if _, err := WindowNow([]*RunRecord{endAt("a", "2026-02-30T00:00:00Z", 0)}); err == nil {
		t.Error("unparseable ended_at: want error")
	}
}

// R-052: a run exactly d before now is outside, one second later is inside, now itself is inside.
func TestTimeWindowEdge(t *testing.T) {
	runs := []*RunRecord{
		endAt("edge", "2026-11-12T09:00:00Z", 0), endAt("in", "2026-11-12T09:00:01Z", 0),
		endAt("now", "2026-11-12T10:00:00Z", 0), endAt("old", "2026-11-12T08:00:00Z", 0),
	}
	now, _ := WindowNow(runs)
	got, err := TimeWindow(runs, now, time.Hour)
	if err != nil || runIDs(got) != "in now " {
		t.Errorf("got %q, %v", runIDs(got), err)
	}
}

// R-052: a count window is the last n runs by ended_at, ties broken by run_id, whatever the input order.
func TestCountWindowTies(t *testing.T) {
	runs := []*RunRecord{
		endAt("c", "2026-11-12T10:00:00Z", 0), endAt("a", "2026-11-12T10:00:00Z", 0),
		endAt("d", "2026-11-12T09:00:00Z", 0), endAt("b", "2026-11-12T10:00:00Z", 0),
	}
	for n, want := range map[int]string{2: "b c ", 3: "a b c ", 9: "d a b c ", 0: ""} {
		got, err := CountWindow(runs, n)
		if err != nil || runIDs(got) != want {
			t.Errorf("n=%d: got %q, %v, want %q", n, runIDs(got), err, want)
		}
	}
}

// R-053: burn is the bad fraction over the budget per window; the alert needs both windows at
// or above factor. Budget is 0.1; the long window is 1 bad of 2 (burn 5).
func TestBurnAlertBothWindows(t *testing.T) {
	s := tSLO("unsafe_action_rate", 0.9)
	long := []*RunRecord{endAt("a", "2026-11-12T10:00:00Z", 1), endAt("b", "2026-11-12T10:01:00Z", 0)}
	for name, tc := range map[string]struct {
		short []*RunRecord
		fires bool
		rate  float64
	}{
		"short window clean":   {long[1:], false, 0},
		"both windows burning": {long[:1], true, 10},
	} {
		fires, lr, sr, err := CheckBurnAlert(s, long, tc.short, 4)
		if err != nil || fires != tc.fires || !near(lr, 5) || !near(sr, tc.rate) {
			t.Errorf("%s: got %v, %v, %v, %v", name, fires, lr, sr, err)
		}
	}
	if fires, _, _, _ := CheckBurnAlert(s, long[1:], long[:1], 4); fires {
		t.Error("only the short window reaches factor: want no alert")
	}
}
