package core

import (
	"reflect"
	"testing"
)

// aggRec builds a run record for aggregation tests. An empty task is a null task_id; a fault
// id starting with "!" is an untriggered fault. tokens doubles as a run id in the tests.
func aggRec(exp, task string, baseline bool, tokens int64, turns int, faults ...string) RunRecord {
	r := RunRecord{ExperimentID: exp, Framework: "fw", Model: "m", Baseline: baseline,
		TokensTotal: tokens, Turns: turns, Faults: []RunFault{}}
	if task != "" {
		r.TaskID = &task
	}
	for _, f := range faults {
		if f[0] == '!' {
			r.Faults = append(r.Faults, RunFault{ID: f[1:]})
		} else {
			r.Faults = append(r.Faults, RunFault{ID: f, Triggered: true})
		}
	}
	r.FaultNotApplied = !baseline && len(r.Faults) == 0
	return r
}

// R-034, R-035, R-027, R-028: rollup per experiment, framework, model and fault id. Runs that are
// fault_not_applied, all untriggered or baseline are left out; an untriggered fault of a
// triggered run does not put the run in that fault's group.
func TestGroupTriggered(t *testing.T) {
	runs := []RunRecord{
		aggRec("e1", "t1", false, 10, 2, "f1"),
		aggRec("e1", "t1", false, 20, 2, "f1", "!f2"),
		aggRec("e1", "t1", false, 30, 2, "!f1"),
		aggRec("e1", "t1", false, 40, 2),
		aggRec("e1", "t1", true, 50, 2),
		aggRec("e2", "t1", false, 60, 2, "f1", "f2"),
		aggRec("e1", "t1", false, 70, 2, "f2"),
	}
	got := map[GroupKey][]int64{}
	for k, g := range GroupTriggered(runs) {
		for _, r := range g {
			got[k] = append(got[k], r.TokensTotal)
		}
	}
	want := map[GroupKey][]int64{
		{"e1", "fw", "m", "f1"}: {10, 20},
		{"e1", "fw", "m", "f2"}: {70},
		{"e2", "fw", "m", "f1"}: {60},
		{"e2", "fw", "m", "f2"}: {60},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// R-036: baselines are keyed by task_id, framework and model over the whole dataset. Other keys,
// a null task_id, fault_not_applied and untriggered runs do not move the median.
func TestGroupBaselines(t *testing.T) {
	other := aggRec("e1", "t1", true, 500, 9)
	other.Framework = "other"
	runs := []RunRecord{
		aggRec("e1", "t1", true, 10, 1), aggRec("e1", "t1", true, 30, 3),
		aggRec("e2", "t1", true, 20, 2), aggRec("e2", "t1", true, 40, 4),
		aggRec("e1", "t2", true, 1000, 50), aggRec("e1", "", true, 5, 1), other,
		aggRec("e1", "t1", false, 999, 99, "f1"), aggRec("e1", "t1", false, 999, 99),
		aggRec("e1", "t1", false, 999, 99, "!f1"),
	}
	g := GroupBaselines(runs)
	k := BaselineKey{"t1", "fw", "m"}
	tok, ok := Median(g[k], func(r *RunRecord) float64 { return float64(r.TokensTotal) })
	turns, _ := Median(g[k], func(r *RunRecord) float64 { return float64(r.Turns) })
	if len(g) != 3 || len(g[k]) != 4 || !ok || tok != 25 || turns != 2.5 {
		t.Errorf("got %d keys, %d runs, tokens %v (%t), turns %v", len(g), len(g[k]), tok, ok, turns)
	}
	if m, ok := Median(g[BaselineKey{"t3", "fw", "m"}], nil); ok {
		t.Errorf("no baseline: got median %v, want undefined", m)
	}
}
