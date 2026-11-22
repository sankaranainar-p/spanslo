package core

import (
	"strings"
	"testing"
	"time"
)

func tokChat(n *Names, id string, ms int, in, out int64) Span {
	s := mkChat(n, id, ms)
	s.Attrs[n.Attributes.InputTokens] = in
	s.Attrs[n.Attributes.OutputTokens] = out
	return s
}

// metricsSpans: chats c1 (0 ms), c2 (1000) and c3 (1300); call f (100 ms, ends at 150) in turn 1
// faults; call ok (1200) is the recovery in turn 2. c1 and c3 lie outside the f to ok interval.
func metricsSpans(n *Names) []Span {
	return []Span{
		mkSpan(n, "r", n.Operations.Run, 0, map[string]any{}),
		tokChat(n, "c1", 0, 1000, 1000), tokChat(n, "c2", 1000, 10, 5), tokChat(n, "c3", 1300, 700, 700),
		recCall(n, "f", "search", 100, true), recCall(n, "ok", "search", 1200, false),
	}
}

// recoverFor finds the qualifying call for last fault span f on "search" and scores the episode.
// The episode's event turn (7) is wrong on purpose: it must be ignored (R-005, R-007).
func recoverFor(t *testing.T, n *Names, spans []Span, f Span) (EpisodeMetrics, []string) {
	t.Helper()
	idx := mustIndex(t, n, spans)
	rec := FindRecovery(idx, f, "search", nil, false)
	if rec == nil {
		t.Fatal("no qualifying recovery call")
	}
	return ComputeEpisodeMetrics(n, idx, spans, Episode{Target: "search", LastFaultTurn: 7}, f, f, rec)
}

// R-017 to R-020: a recovered episode; the interval ends at the recovery call.
func TestEpisodeMetricsRecovered(t *testing.T) {
	n := idxNames(t)
	spans := metricsSpans(n)
	m, warns := recoverFor(t, n, spans, spans[4])
	if m.TurnsToRecover == nil || *m.TurnsToRecover != 1 {
		t.Errorf("turns_to_recover: want 1 (chat derived, event says 7), got %+v", m)
	}
	if m.TimeToRecoverMS == nil || *m.TimeToRecoverMS != 1050 {
		t.Errorf("time_to_recover_ms: want 1050, got %+v", m)
	}
	if m.CascadeSpans != 3 || m.RecoveryTokens != 15 || len(warns) != 0 {
		t.Errorf("got %+v, warnings %v, want 3 spans, 15 tokens, no warnings", m, warns)
	}
}

// R-017 to R-020: an unrecovered episode has nulls; its interval runs to the end of the run.
func TestEpisodeMetricsUnrecovered(t *testing.T) {
	n := idxNames(t)
	spans := metricsSpans(n)[:5] // without the recovery call
	f := spans[4]
	idx := mustIndex(t, n, spans)
	if FindRecovery(idx, f, "search", nil, false) != nil {
		t.Fatal("unexpected recovery call")
	}
	m, warns := ComputeEpisodeMetrics(n, idx, spans, Episode{Target: "search"}, f, f, nil)
	if m.TurnsToRecover != nil || m.TimeToRecoverMS != nil {
		t.Errorf("want null turns and time, got %+v", m)
	}
	if m.CascadeSpans != 3 || m.RecoveryTokens != 1415 || len(warns) != 0 {
		t.Errorf("got %+v, warnings %v, want 3 spans, 1415 tokens, no warnings", m, warns)
	}
}

// R-018: a recovery call that starts before the last fault span ended clamps to 0 and warns.
func TestEpisodeMetricsNegativeTimeClamps(t *testing.T) {
	n := idxNames(t)
	spans := metricsSpans(n)
	f := spans[4]
	f.End = f.Start.Add(2 * time.Second) // ends at 2100 ms, after ok starts at 1200 ms
	spans[4] = f
	idx := mustIndex(t, n, spans)
	rec := &Call{Span: spans[5], Tool: "search", T: 2}
	m, warns := ComputeEpisodeMetrics(n, idx, spans, Episode{Target: "search"}, f, f, rec)
	if m.TimeToRecoverMS == nil || *m.TimeToRecoverMS != 0 {
		t.Errorf("time_to_recover_ms: want 0, got %+v", m)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "clamped") {
		t.Errorf("want one clamp warning, got %v", warns)
	}
}

// R-018: an unfinished fault span (zero End) measures from its start, as FindRecovery does.
func TestEpisodeMetricsZeroEndFault(t *testing.T) {
	n := idxNames(t)
	spans := metricsSpans(n)
	spans[4].End = time.Time{}
	m, warns := recoverFor(t, n, spans, spans[4])
	if m.TimeToRecoverMS == nil || *m.TimeToRecoverMS != 1100 || len(warns) != 0 {
		t.Errorf("want 1100 ms and no warnings, got %+v, %v", m, warns)
	}
}

// R-005, R-007: a last fault on a span outside the index (the root) has no derived turn, so
// turns_to_recover is null with a warning; time to recover is still measured.
func TestEpisodeMetricsFaultOutsideIndex(t *testing.T) {
	n := idxNames(t)
	spans := metricsSpans(n)
	root := spans[0]
	idx := mustIndex(t, n, spans)
	rec := FindRecovery(idx, root, "search", nil, false)
	if rec == nil {
		t.Fatal("no qualifying recovery call")
	}
	m, warns := ComputeEpisodeMetrics(n, idx, spans, Episode{Target: "search", LastFaultTurn: 1}, spans[4], root, rec)
	if m.TurnsToRecover != nil || m.TimeToRecoverMS == nil || *m.TimeToRecoverMS != 1200 {
		t.Errorf("want null turns and 1200 ms, got %+v", m)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "outside the index") {
		t.Errorf("want one outside the index warning, got %v", warns)
	}
}

// R-017: the last fault is on a chat span (model surface); its own turn 2 is subtracted from 3.
func TestEpisodeMetricsFaultOnChatSpan(t *testing.T) {
	n := idxNames(t)
	spans := []Span{
		mkSpan(n, "r", n.Operations.Run, 0, map[string]any{}),
		tokChat(n, "c1", 0, 1000, 1000), tokChat(n, "c2", 1000, 10, 5), tokChat(n, "c3", 1300, 700, 700),
		recCall(n, "ok", "search", 1400, false),
	}
	m, _ := recoverFor(t, n, spans, spans[2])
	if m.TurnsToRecover == nil || *m.TurnsToRecover != 1 || m.TimeToRecoverMS == nil || *m.TimeToRecoverMS != 400 {
		t.Errorf("want 1 turn and 400 ms, got %+v", m)
	}
}
