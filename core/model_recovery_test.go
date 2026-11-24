package core

import (
	"reflect"
	"testing"
	"time"
)

// modelFault adds a triggered model surface fault event on target "gpt" to a span.
func modelFault(s Span, turn int64) Span {
	e := faultEvent(true, turn)
	e.Attrs[faultTargetAttr] = "gpt"
	e.Attrs[faultSurfaceAttr] = "model"
	s.Events = append(s.Events, e)
	return s
}

// errChat is a chat span with an error status.
func errChat(n *Names, id string, ms int) Span {
	s := mkChat(n, id, ms)
	s.End = s.Start.Add(50 * time.Millisecond)
	s.Status.Code = StatusError
	return s
}

// R-012, R-013: model surface recovery. Chats start at 0, 1000 and 2000 ms.
func TestFindRecoveryModelSurface(t *testing.T) {
	n := idxNames(t)
	ok := func(id string, ms int) Span { return mkChat(n, id, ms) }
	fb := []string{"backup"}
	tool := recCall(n, "k", "search", 100, true)
	for _, tc := range []struct {
		name, surface, fault, want string
		fallbacks                  []string
		persistent                 bool
		wantT                      int
		spans                      []Span
	}{
		{"later success", "model", "c1", "c2", nil, false, 2, []Span{errChat(n, "c1", 0), ok("c2", 1000)}},
		{"skips errored chat", "model", "c1", "c3", nil, false, 3, []Span{errChat(n, "c1", 0), errChat(n, "c2", 1000), ok("c3", 2000)}},
		{"later chat error", "model", "c1", "", nil, false, 0, []Span{errChat(n, "c1", 0), ok("c2", 1000), errChat(n, "c3", 2000)}},
		{"no later chat", "model", "c1", "", nil, false, 0, []Span{errChat(n, "c1", 0)}},
		{"persistent, no fallback", "model", "c1", "", nil, true, 0, []Span{errChat(n, "c1", 0), ok("c2", 1000)}},
		{"persistent, fallback", "model", "c1", "fb", fb, true, 2, []Span{errChat(n, "c1", 0), ok("c2", 1000), recCall(n, "fb", "backup", 1500, false)}},
		{"fault on tool call span", "model", "k", "c2", nil, false, 2, []Span{ok("c1", 0), tool, ok("c2", 1000)}},
		{"tool span fault, later chat error", "model", "k", "", nil, false, 0, []Span{ok("c1", 0), tool, ok("c2", 1000), errChat(n, "c3", 2000)}},
		{"tool span fault, persistent, no fallback", "model", "k", "", nil, true, 0, []Span{ok("c1", 0), tool, ok("c2", 1000)}},
		{"tool surface ignores chats", "tool", "k", "", nil, false, 0, []Span{ok("c1", 0), tool, ok("c2", 1000)}},
	} {
		idx := mustIndex(t, n, tc.spans)
		var f Span
		for _, s := range tc.spans {
			if s.SpanID == tc.fault {
				f = s
			}
		}
		got, gotT := "", 0
		if r := FindRecoveryFor(idx, f, tc.surface, "gpt", tc.fallbacks, tc.persistent); r != nil {
			got, gotT = r.Span.SpanID, r.T
		}
		if got != tc.want || gotT != tc.wantT {
			t.Errorf("%s: got %q turn %d, want %q turn %d", tc.name, got, gotT, tc.want, tc.wantT)
		}
	}
}

// R-010: two model faults on one chat span are one episode; a successful chat between two
// faults splits them. Faults on tool call spans group the same way.
func TestGroupEpisodesModelSurface(t *testing.T) {
	n := idxNames(t)
	root := rootWith(n, map[string]any{ModeAttr: "single"})
	tool := func(id string, ms int, turn int64) Span {
		return modelFault(recCall(n, id, "search", ms, true), turn)
	}
	for name, tc := range map[string]struct {
		spans []Span
		want  []Episode
	}{
		"two faults, one chat span": {[]Span{root, modelFault(modelFault(errChat(n, "c1", 0), 1), 1), mkChat(n, "c2", 1000)},
			[]Episode{{"gpt", 1, 1, 2}}},
		"success between faults": {[]Span{root, modelFault(errChat(n, "c1", 0), 1), mkChat(n, "c2", 1000), modelFault(errChat(n, "c3", 2000), 3)},
			[]Episode{{"gpt", 1, 1, 1}, {"gpt", 3, 3, 1}}},
		"error chat between faults": {[]Span{root, modelFault(errChat(n, "c1", 0), 1), modelFault(errChat(n, "c2", 1000), 2)},
			[]Episode{{"gpt", 1, 2, 2}}},
		"tool span faults, success between": {[]Span{root, mkChat(n, "c1", 0), tool("k1", 100, 1), mkChat(n, "c2", 1000), tool("k2", 1100, 2)},
			[]Episode{{"gpt", 1, 1, 1}, {"gpt", 2, 2, 1}}},
		"tool span faults, error chat between": {[]Span{root, mkChat(n, "c1", 0), tool("k1", 100, 1), errChat(n, "c2", 1000), tool("k2", 1100, 2)},
			[]Episode{{"gpt", 1, 2, 2}}},
	} {
		f, err := ExtractFaults(tc.spans)
		if err != nil {
			t.Fatal(err)
		}
		if got := GroupEpisodes(mustIndex(t, n, tc.spans), tc.spans, f.TriggeredFaults(), nil, false); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %+v, want %+v", name, got, tc.want)
		}
	}
}

// R-012, R-017: a model surface episode recovered end to end through ScoreRun. The fault is on
// chat c1 (turn 1), chat c2 (turn 2) succeeds, so turns_to_recover is 1.
func TestScoreRunModelSurfaceRecovered(t *testing.T) {
	n := idxNames(t)
	c1 := modelFault(tokChat(n, "c1", 0, 10, 5), 1)
	c1.End = c1.Start.Add(50 * time.Millisecond)
	c1.Status.Code = StatusError
	r, warns := mustScore(t, n, []Span{scoreRoot(n), c1, tokChat(n, "c2", 1000, 20, 7)})
	want := []RunEpisode{{"gpt", 1, 1, 1, true, ptr(1), ptr(int64(950)), 3, 42}}
	if !reflect.DeepEqual(r.Episodes, want) || tri(r.TaskVerifiedRecovery) != "true" || len(warns) != 0 {
		t.Errorf("got %+v, tvr %s, warnings %v, want %+v", r.Episodes, tri(r.TaskVerifiedRecovery), warns, want)
	}
}
