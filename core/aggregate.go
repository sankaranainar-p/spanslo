package core

import "slices"

// GroupKey is one rollup cell of the fault SLIs (R-034).
type GroupKey struct{ ExperimentID, Framework, Model, FaultID string }

// BaselineKey keys baseline runs (R-036).
type BaselineKey struct{ TaskID, Framework, Model string }

// IsTriggered reports whether a run counts in the fault SLIs: not a baseline, not
// fault_not_applied, and at least one triggered fault event (R-027, R-028, R-035).
func IsTriggered(r *RunRecord) bool {
	if r.Baseline || r.FaultNotApplied {
		return false
	}
	return slices.ContainsFunc(r.Faults, func(f RunFault) bool { return f.Triggered })
}

// GroupTriggered rolls triggered runs up per experiment_id, framework, model and fault id
// (R-034, R-035). A run appears once in the group of each distinct fault id it triggered;
// an untriggered fault does not put it in that fault's group. Runs keep input order.
func GroupTriggered(runs []RunRecord) map[GroupKey][]*RunRecord {
	out := map[GroupKey][]*RunRecord{}
	for i := range runs {
		r := &runs[i]
		if !IsTriggered(r) {
			continue
		}
		seen := map[string]bool{}
		for _, f := range r.Faults {
			if !f.Triggered || seen[f.ID] {
				continue
			}
			seen[f.ID] = true
			k := GroupKey{r.ExperimentID, r.Framework, r.Model, f.ID}
			out[k] = append(out[k], r)
		}
	}
	return out
}

// GroupBaselines collects baseline runs by task_id, framework and model over the whole
// dataset (R-036). A baseline with a null task_id has no key and is left out.
// ponytail: null task_id baselines are dropped; R-044 and R-045 need a task_id anyway.
func GroupBaselines(runs []RunRecord) map[BaselineKey][]*RunRecord {
	out := map[BaselineKey][]*RunRecord{}
	for i := range runs {
		r := &runs[i]
		if !r.Baseline || r.TaskID == nil {
			continue
		}
		k := BaselineKey{*r.TaskID, r.Framework, r.Model}
		out[k] = append(out[k], r)
	}
	return out
}

// Median is the median of val over runs; even counts average the two middle values.
// ok is false for no runs: the median is undefined (R-066).
func Median(runs []*RunRecord, val func(*RunRecord) float64) (float64, bool) {
	if len(runs) == 0 {
		return 0, false
	}
	v := make([]float64, len(runs))
	for i, r := range runs {
		v[i] = val(r)
	}
	slices.Sort(v)
	m := len(v) / 2
	if len(v)%2 == 1 {
		return v[m], true
	}
	return (v[m-1] + v[m]) / 2, true
}
