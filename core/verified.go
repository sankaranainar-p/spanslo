package core

// GracefulErrorAttr is the oracle attribute on the task success event (R-026). Like ModeAttr it
// is a spanslo name, not a gen_ai name.
const GracefulErrorAttr = "spanslo.oracle.graceful_error"

// TaskVerifiedRecovery is the run level recovery verdict (R-015). recovered holds one entry per
// triggered episode. It is nil for baseline runs and runs with no triggered episode; otherwise it
// is true only when every episode recovered and the oracle passed.
func TaskVerifiedRecovery(baseline bool, recovered []bool, success bool) *bool {
	if baseline || len(recovered) == 0 {
		return nil
	}
	v := AllRecovered(recovered) && success
	return &v
}

// GracefulError reads the oracle's graceful_error flag from the task success event. It is set
// only when at least one episode is not recovered; otherwise, or when the flag is absent or not
// a boolean, it is nil (R-026).
func GracefulError(n *Names, spans []Span, recovered []bool) *bool {
	if AllRecovered(recovered) {
		return nil
	}
	er := n.Events.EvaluationResult
	for _, s := range spans {
		for _, e := range s.Events {
			if e.Name != er.Name || e.Attrs[er.Attrs.EvaluationName] != er.SuccessName {
				continue
			}
			if v, ok := e.Attrs[GracefulErrorAttr].(bool); ok {
				return &v
			}
			return nil
		}
	}
	return nil
}
