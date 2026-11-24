package core

import "testing"

// w47 builds a run record that ended on the given day of November 2026.
func w47(id, task, day string, baseline, ok bool, tokens int64, faults ...string) *RunRecord {
	r := aggRec("e", task, baseline, tokens, 1, faults...)
	r.RunID, r.EndedAt, r.Success = id, "2026-11-"+day+"T00:00:00Z", ok
	return &r
}

// R-036, R-052: baselines come from the whole dataset, so a window that drops the older
// baseline runs gives the same TAF and resilience ratio; each run meets the baseline of its
// own task; a run with no matching baseline is undefined.
func TestBaselinesFromWholeDataset(t *testing.T) {
	all := []*RunRecord{
		w47("b1", "t1", "01", true, true, 100), w47("b2", "t1", "02", true, true, 100),
		w47("b3", "t1", "03", true, false, 200),
		w47("c1", "t2", "04", true, true, 1000), w47("c2", "t2", "05", true, true, 1000),
		w47("f1", "t1", "10", false, true, 300, "f1"),
		w47("f2", "t2", "11", false, true, 1500, "f1"),
		w47("f3", "t3", "12", false, false, 50, "f1"),
	}
	win, err := CountWindow(all, 3)
	if err != nil || len(win) != 3 {
		t.Fatalf("window: %v, %d runs", err, len(win))
	}
	rrSLO, tafSLO := tSLO("resilience_ratio", 1.0), thrSLO("taf", 2, 0.5)
	for _, s := range []SLO{rrSLO, tafSLO} {
		full, err1 := EvaluateSLOIn(s, all, all)
		cut, err2 := EvaluateSLOIn(s, win, all)
		if err1 != nil || err2 != nil || *full != *cut {
			t.Errorf("%s: window changed the result: %+v vs %+v (%v, %v)", s.SLI, full, cut, err1, err2)
		}
	}
	rr, _ := EvaluateSLOIn(rrSLO, win, all)
	// f1 against t1 (S0 2/3), f2 against t2 (S0 1); f3 has no baseline and is left out
	if !rr.Defined || !near(rr.Value.Value, 1.2) {
		t.Errorf("resilience ratio: got %+v, want 1.2", rr.Value)
	}
	taf, _ := EvaluateSLOIn(tafSLO, win, all)
	// f1 is 300 over t1's median 100, f2 is 1500 over t2's median 1000, f3 is undefined
	if taf.Total != 2 || taf.Bad != 1 {
		t.Errorf("taf: got %d bad of %d, want 1 of 2", taf.Bad, taf.Total)
	}
	none, _ := EvaluateSLOIn(tafSLO, win[2:], all)
	if none.Defined || none.Total != 0 {
		t.Errorf("no matching baseline: got %+v, want undefined", none)
	}
}

// R-036: a burn alert uses baselines from the whole dataset. The windows hold only triggered
// runs, so CheckBurnAlert finds no baseline and never fires; CheckBurnAlertIn does.
func TestCheckBurnAlertInUsesDataset(t *testing.T) {
	base := []*RunRecord{w47("b1", "t1", "01", true, true, 100), w47("b2", "t1", "02", true, true, 100)}
	long := []*RunRecord{w47("f1", "t1", "10", false, true, 300, "f1"), w47("f2", "t1", "11", false, true, 300, "f1")}
	short := long[1:]
	dataset := append(append([]*RunRecord{}, base...), long...)
	s := thrSLO("taf", 2, 0.5) // every taf is 3: bad fraction 1 over budget 0.5 is burn 2
	fires, lr, sr, err := CheckBurnAlertIn(s, long, short, dataset, 2)
	if err != nil || !fires || !near(lr, 2) || !near(sr, 2) {
		t.Errorf("with dataset: got %v, %v, %v, %v, want fires with burn 2 and 2", fires, lr, sr, err)
	}
	fires, lr, sr, err = CheckBurnAlert(s, long, short, 2)
	if err != nil || fires || lr != 0 || sr != 0 {
		t.Errorf("windows only: got %v, %v, %v, %v, want no alert and burn 0", fires, lr, sr, err)
	}
	if fires, _, _, _ = CheckBurnAlertIn(s, long, short, dataset, 2.5); fires {
		t.Error("factor 2.5: alert fired with burn below factor")
	}
}
