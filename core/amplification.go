package core

// amplification is the run's value over the median value of the baseline runs with the same
// task_id, framework and model (R-044, R-045). It is undefined, never computed, when task_id is
// null, no baseline has the key, or the median is 0 (R-066).
func amplification(r *RunRecord, baselines map[BaselineKey][]*RunRecord, val func(*RunRecord) float64) Ratio {
	if r.TaskID == nil {
		return Ratio{}
	}
	m, ok := Median(baselines[BaselineKey{*r.TaskID, r.Framework, r.Model}], val)
	if !ok || m == 0 {
		return Ratio{}
	}
	return Ratio{val(r) / m, true}
}

// TAF is the Token Amplification Factor: tokens_total / baseline median tokens_total (R-044).
func TAF(r *RunRecord, baselines map[BaselineKey][]*RunRecord) Ratio {
	return amplification(r, baselines, func(x *RunRecord) float64 { return float64(x.TokensTotal) })
}

// TurnAmplification is turns / baseline median turns (R-045).
func TurnAmplification(r *RunRecord, baselines map[BaselineKey][]*RunRecord) Ratio {
	return amplification(r, baselines, func(x *RunRecord) float64 { return float64(x.Turns) })
}
