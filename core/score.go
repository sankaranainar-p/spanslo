package core

import (
	"errors"
	"slices"
)

// RunFault is one entry of the faults array of a run record (R-057). Virtual is true when
// the fault declared a simulated timeout duration.
type RunFault struct {
	ID        string `json:"id"`
	Turn      int64  `json:"turn"`
	CallIndex int64  `json:"call_index"`
	Triggered bool   `json:"triggered"`
	Virtual   bool   `json:"virtual"`
}

// RunEpisode is one entry of the episodes array of a run record (R-011).
type RunEpisode struct {
	Target           string `json:"target"`
	FirstFaultTurn   int64  `json:"first_fault_turn"`
	LastFaultTurn    int64  `json:"last_fault_turn"`
	FaultCount       int    `json:"fault_count"`
	SurfaceRecovered bool   `json:"surface_recovered"`
	TurnsToRecover   *int   `json:"turns_to_recover"`
	TimeToRecoverMS  *int64 `json:"time_to_recover_ms"`
	CascadeSpans     int    `json:"cascade_spans"`
	RecoveryTokens   int64  `json:"recovery_tokens"`
}

// RunRecord is the run record (R-031). The JSON tags mirror spec/run-record.schema.json;
// nullable fields are pointers and have no omitempty, so every required key is written.
type RunRecord struct {
	RunID                 string       `json:"run_id"`
	ExperimentID          string       `json:"experiment_id"`
	TaskID                *string      `json:"task_id"`
	TaskTurnsExpected     *int         `json:"task_turns_expected"`
	Framework             string       `json:"framework"`
	Model                 string       `json:"model"`
	Baseline              bool         `json:"baseline"`
	Mode                  string       `json:"mode"`
	Faults                []RunFault   `json:"faults"`
	Episodes              []RunEpisode `json:"episodes"`
	Success               bool         `json:"success"`
	TaskVerifiedRecovery  *bool        `json:"task_verified_recovery"`
	Turns                 int          `json:"turns"`
	ModelCalls            int          `json:"model_calls"`
	ToolCalls             int          `json:"tool_calls"`
	TokensTotal           int64        `json:"tokens_total"`
	UnsafeActions         int          `json:"unsafe_actions"`
	UntaggedToolCalls     int          `json:"untagged_tool_calls"`
	RetryBudgetViolations int          `json:"retry_budget_violations"`
	GracefulError         *bool        `json:"graceful_error"`
	FaultNotApplied       bool         `json:"fault_not_applied"`
	TurnMismatches        int          `json:"turn_mismatches"`
	EndedAt               string       `json:"ended_at"`
	SemconvVersion        string       `json:"semconv_version"`
}

// asInvalid turns an error from a scoring step into an invalid run reason.
func asInvalid(err error) *InvalidRun {
	var inv *InvalidRun
	if errors.As(err, &inv) {
		return inv
	}
	return &InvalidRun{err.Error()}
}

// firstString returns the first non empty string value of key over spans, which the caller
// has sorted by (start, span id), so the answer does not depend on input order.
func firstString(spans []Span, key string) string {
	for _, s := range spans {
		if v, ok := s.Attrs[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// rootSpan is the run's root: the run span with no parent that sorts first. It is the same
// rule as ParseOutcome's, and nil when there is none.
func rootSpan(n *Names, spans []Span) *Span {
	var root *Span
	for i := range spans {
		s := &spans[i]
		if op, _ := s.Attrs[n.Attributes.OperationName].(string); op != n.Operations.Run || s.ParentSpanID != "" {
			continue
		}
		if root == nil || spanCmp(*s, *root) < 0 {
			root = s
		}
	}
	return root
}

// ScoreRun scores the spans of one run into a run record (R-030, R-073 to R-081). Identity
// comes from span attributes; c is the effective contract of the run's fault (R-025). It
// returns the record and warnings for the caller to print (R-018), or an *InvalidRun with a
// nil record (R-016, R-080). It does no I/O (R-008).
func ScoreRun(n *Names, c *Contract, spans []Span) (*RunRecord, []string, *InvalidRun) {
	out, err := ParseOutcome(n, spans)
	if err != nil {
		return nil, nil, asInvalid(err)
	}
	idx, err := IndexRun(n, spans)
	if err != nil {
		return nil, nil, asInvalid(err)
	}
	info, err := ExtractFaults(spans)
	if err != nil {
		return nil, nil, asInvalid(err)
	}
	if err := DeriveFaultTurns(idx, info); err != nil {
		return nil, nil, asInvalid(err)
	}
	sorted := slices.Clone(spans)
	slices.SortStableFunc(sorted, spanCmp)
	id := n.Spanslo.RunIdentity
	r := &RunRecord{
		Baseline: info.Baseline, Mode: info.Mode, Success: out.Success, EndedAt: out.EndedAt,
		SemconvVersion: n.Semconv.Commit, FaultNotApplied: info.FaultNotApplied,
		Faults: []RunFault{}, Episodes: []RunEpisode{},
		Turns: len(idx.Chats), ModelCalls: len(idx.Chats), ToolCalls: len(idx.Calls),
		TurnMismatches: TurnMismatches(idx, spans),
	}
	if r.RunID = firstString(sorted, id.RunID); r.RunID == "" {
		for _, s := range sorted {
			if r.RunID = s.TraceID; r.RunID != "" {
				break
			}
		}
	}
	if r.RunID == "" {
		return nil, nil, &InvalidRun{"run has no " + id.RunID + " and no trace id"}
	}
	for _, a := range []struct {
		key string
		dst *string
	}{{id.ExperimentID, &r.ExperimentID}, {id.Framework, &r.Framework}, {id.Model, &r.Model}} {
		if *a.dst = firstString(sorted, a.key); *a.dst == "" {
			return nil, nil, &InvalidRun{"attribute " + a.key + " is missing or empty"}
		}
	}
	if v := firstString(sorted, id.TaskID); v != "" {
		r.TaskID = &v
	}
	root := rootSpan(n, spans)
	if root == nil {
		return nil, nil, &InvalidRun{"no root " + n.Operations.Run + " span"}
	}
	if v, ok := root.Attrs[n.Spanslo.Task.TurnsExpected].(int64); ok && v >= 1 {
		t := int(v)
		r.TaskTurnsExpected = &t
	}
	for _, ch := range idx.Chats {
		r.TokensTotal += tokenCount(ch.Attrs[n.Attributes.InputTokens]) + tokenCount(ch.Attrs[n.Attributes.OutputTokens])
	}
	cr := ScoreContracts(n, idx, c)
	r.UnsafeActions, r.UntaggedToolCalls, r.RetryBudgetViolations = cr.UnsafeActions, cr.UntaggedToolCalls, cr.RetryBudgetViolations
	for _, f := range info.Faults {
		r.Faults = append(r.Faults, RunFault{f.ID, f.Turn, f.CallIndex, f.Triggered, f.VirtualDurationMS > 0})
	}
	eps, recovered, warns := scoreEpisodes(n, idx, spans, info)
	r.Episodes = eps
	r.TaskVerifiedRecovery = TaskVerifiedRecovery(info.Baseline, recovered, out.Success)
	r.GracefulError = GracefulError(n, spans, recovered)
	return r, warns, nil
}

// scoreEpisodes assembles the episodes of a run and whether each recovered. Recovery is judged
// on the calls before the next fault of the same target, so that fault's own error is not a
// later error (R-012).
func scoreEpisodes(n *Names, idx *RunIndex, spans []Span, info *FaultInfo) ([]RunEpisode, []bool, []string) {
	byID := make(map[string]Span, len(spans))
	for _, s := range spans {
		byID[s.SpanID] = s
	}
	trig := info.TriggeredFaults()
	byTarget := map[string][]FaultEvent{}
	for _, f := range trig {
		byTarget[f.Target] = append(byTarget[f.Target], f)
	}
	fb := FallbackTools(n, spans)
	persistent := info.Mode == "persistent"
	used := map[string]int{}
	out := []RunEpisode{}
	var recovered []bool
	var warns []string
	for _, e := range GroupEpisodes(idx, spans, trig, fb, persistent) {
		fs, k := byTarget[e.Target], used[e.Target]
		end := k + e.FaultCount
		used[e.Target] = end
		first, last := byID[fs[k].SpanID], byID[fs[end-1].SpanID]
		win := idx
		if end < len(fs) {
			next := byID[fs[end].SpanID]
			win = &RunIndex{Chats: idx.Chats}
			for _, c := range idx.Calls {
				if c.Span.SpanID != next.SpanID && c.Span.Start.Before(next.Start) {
					win.Calls = append(win.Calls, c)
				}
			}
		}
		rec := FindRecovery(win, last, e.Target, fb, persistent)
		m, w := ComputeEpisodeMetrics(n, idx, spans, e, first, last, rec)
		warns = append(warns, w...)
		recovered = append(recovered, rec != nil)
		out = append(out, RunEpisode{e.Target, e.FirstFaultTurn, e.LastFaultTurn, e.FaultCount, rec != nil,
			m.TurnsToRecover, m.TimeToRecoverMS, m.CascadeSpans, m.RecoveryTokens})
	}
	return out, recovered, warns
}
