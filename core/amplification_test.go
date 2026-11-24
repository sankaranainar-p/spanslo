package core

import "testing"

// R-044, R-045, R-066: a computed case and each undefined case. The t1 baselines have median
// tokens 20 and median turns 3; the t0 baseline has median 0. A want of 0 means undefined.
func TestAmplification(t *testing.T) {
	b := GroupBaselines([]RunRecord{
		aggRec("e", "t1", true, 10, 2), aggRec("e", "t1", true, 30, 4),
		aggRec("e", "t0", true, 0, 0),
	})
	otherFw := aggRec("e", "t1", false, 50, 6, "f1")
	otherFw.Framework = "other"
	for name, tc := range map[string]struct {
		r         RunRecord
		taf, turn float64
	}{
		"computed":                  {aggRec("e", "t1", false, 50, 6, "f1"), 2.5, 2},
		"equal to median":           {aggRec("e", "t1", false, 20, 3, "f1"), 1, 1},
		"null task_id":              {aggRec("e", "", false, 50, 6, "f1"), 0, 0},
		"no baseline for task":      {aggRec("e", "t2", false, 50, 6, "f1"), 0, 0},
		"no baseline for framework": {otherFw, 0, 0},
		"median 0":                  {aggRec("e", "t0", false, 5, 5, "f1"), 0, 0},
	} {
		for i, got := range []Ratio{TAF(&tc.r, b), TurnAmplification(&tc.r, b)} {
			want := []float64{tc.taf, tc.turn}[i]
			if got.OK != (want != 0) || (got.OK && !near(got.Value, want)) {
				t.Errorf("%s [%d]: got %+v, want %v", name, i, got, want)
			}
		}
	}
	if got := TAF(&otherFw, nil); got.OK {
		t.Errorf("no baselines: got %+v, want undefined", got)
	}
}
