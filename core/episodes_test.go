package core

import (
	"reflect"
	"testing"
)

// faultOn attaches a triggered fault event for target in the given turn to a call span.
func faultOn(s Span, target string, turn int64) Span {
	e := faultEvent(true, turn)
	e.Attrs[faultTargetAttr] = target
	s.Events = []Event{e}
	return s
}

// episodeFixture indexes a run with chats c1 (0 ms) and c2 (250 ms) around the given calls.
func episodeFixture(t *testing.T, calls []Span) (*RunIndex, []Span) {
	t.Helper()
	n := idxNames(t)
	spans := append([]Span{rootWith(n, map[string]any{ModeAttr: "single"}),
		mkChat(n, "c1", 0), mkChat(n, "c2", 250)}, calls...)
	return mustIndex(t, n, spans), spans
}

// runEpisodes extracts the triggered faults from the calls' events and groups them.
func runEpisodes(t *testing.T, fallbacks []string, persistent bool, calls ...Span) []Episode {
	t.Helper()
	idx, spans := episodeFixture(t, calls)
	f, err := ExtractFaults(spans)
	if err != nil {
		t.Fatal(err)
	}
	return GroupEpisodes(idx, spans, f.TriggeredFaults(), fallbacks, persistent)
}

// R-010, R-011: faults f1, f2 and f3 hit "search"; the calls in mid fall between f2 and f3
// and decide whether the episode breaks. Chat c2 starts at 250: f1 and f2 are turn 1, f3 turn 2.
func TestGroupEpisodes(t *testing.T) {
	n := idxNames(t)
	two := []Episode{{"search", 1, 1, 2}, {"search", 2, 2, 1}}
	one := []Episode{{"search", 1, 2, 3}}
	ok := func(id, tool string, ms int) Span { return recCall(n, id, tool, ms, false) }
	bad := func(id, tool string, ms int) Span { return recCall(n, id, tool, ms, true) }
	fb := []string{"backup"}
	for name, tc := range map[string]struct {
		mid        []Span
		fallbacks  []string
		persistent bool
		want       []Episode
	}{
		"target recovers":              {[]Span{ok("m", "search", 300)}, nil, false, two},
		"fallback recovers":            {[]Span{ok("m", "backup", 300)}, fb, false, two},
		"unrelated call":               {[]Span{ok("m", "other", 300)}, nil, false, one},
		"persistent target no recover": {[]Span{ok("m", "search", 300)}, nil, true, one},
		"persistent fallback recovers": {[]Span{ok("m", "backup", 300)}, fb, true, two},
		"errored call on target":       {[]Span{bad("m", "search", 300)}, nil, false, one},
		"errored call on fallback":     {[]Span{bad("m", "backup", 300)}, fb, false, one},
		"error then success":           {[]Span{bad("m1", "search", 300), ok("m2", "search", 350)}, nil, false, two},
		"success then error":           {[]Span{ok("m1", "search", 300), bad("m2", "search", 350)}, nil, false, one},
	} {
		calls := append([]Span{
			faultOn(bad("f1", "search", 100), "search", 1),
			faultOn(bad("f2", "search", 200), "search", 1),
			faultOn(bad("f3", "search", 400), "search", 2),
		}, tc.mid...)
		if got := runEpisodes(t, tc.fallbacks, tc.persistent, calls...); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %+v, want %+v", name, got, tc.want)
		}
	}
}

// R-010: interleaved faults on "search" and "fetch" split per target and come out in order of
// each target's first fault (not alphabetical). A success on fetch ends only fetch's episode.
func TestGroupEpisodesTwoTargets(t *testing.T) {
	n := idxNames(t)
	got := runEpisodes(t, nil, false,
		faultOn(recCall(n, "s1", "search", 100, true), "search", 1),
		faultOn(recCall(n, "x1", "fetch", 150, true), "fetch", 1),
		recCall(n, "okf", "fetch", 300, false),
		faultOn(recCall(n, "s2", "search", 400, true), "search", 2),
		faultOn(recCall(n, "x2", "fetch", 500, true), "fetch", 2),
	)
	want := []Episode{{"search", 1, 2, 2}, {"fetch", 1, 1, 1}, {"fetch", 2, 2, 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// R-028: an untriggered fault passed in directly never starts or joins an episode.
func TestGroupEpisodesSkipsUntriggered(t *testing.T) {
	n := idxNames(t)
	idx, spans := episodeFixture(t, []Span{
		recCall(n, "f1", "search", 100, true), recCall(n, "f2", "search", 200, true)})
	faults := []FaultEvent{
		{SpanID: "f1", Target: "search", Turn: 1, Triggered: true},
		{SpanID: "f1", Target: "search", Turn: 1},
		{SpanID: "f1", Target: "fetch", Turn: 1},
		{SpanID: "f2", Target: "search", Turn: 1, Triggered: true},
	}
	want := []Episode{{"search", 1, 1, 2}}
	if got := GroupEpisodes(idx, spans, faults, nil, false); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if got := GroupEpisodes(idx, spans, faults[1:3], nil, false); len(got) != 0 {
		t.Errorf("all untriggered: got %+v, want none", got)
	}
}
