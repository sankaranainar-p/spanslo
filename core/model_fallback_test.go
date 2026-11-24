package core

import "testing"

// modelRecovery returns the id of the qualifying recovery for a model surface fault on the
// first chat span (turn 1, target "gpt"), or "".
func modelRecovery(t *testing.T, n *Names, fallbacks []string, persistent bool, chats []Span, calls ...Span) string {
	t.Helper()
	idx := mustIndex(t, n, append(chats, calls...))
	if r := FindRecoveryFor(idx, idx.Chats[0], "model", "gpt", fallbacks, persistent); r != nil {
		return r.Span.SpanID
	}
	return ""
}

// R-012, R-013: a model surface fault also recovers through a successful declared fallback call.
func TestFindRecoveryModelFallback(t *testing.T) {
	n := idxNames(t)
	fb := []string{"backup"}
	badChats := func() []Span { return []Span{mkChat(n, "c1", 0), errChat(n, "c2", 1000), errChat(n, "c3", 2000)} }
	okChats := func() []Span { return []Span{mkChat(n, "c1", 0), mkChat(n, "c2", 1000), mkChat(n, "c3", 2000)} }
	for _, tc := range []struct {
		name       string
		fallbacks  []string
		persistent bool
		chats      []Span
		calls      []Span
		want       string
	}{
		{"fallback, no good chat", fb, false, badChats(), []Span{recCall(n, "ok", "backup", 1500, false)}, "ok"},
		{"fallback errors", fb, false, badChats(), []Span{recCall(n, "e", "backup", 1500, true)}, ""},
		{"later fallback error", fb, false, badChats(), []Span{recCall(n, "ok", "backup", 1200, false), recCall(n, "e", "backup", 1800, true)}, ""},
		{"fallback not declared", nil, false, badChats(), []Span{recCall(n, "ok", "backup", 1500, false)}, ""},
		{"no fallback, chat recovers", nil, false, okChats(), nil, "c2"},
		{"earlier fallback wins", fb, false, okChats(), []Span{recCall(n, "ok", "backup", 500, false)}, "ok"},
		{"earlier chat wins", fb, false, okChats(), []Span{recCall(n, "ok", "backup", 1500, false)}, "c2"},
		{"persistent, fallback only", fb, true, okChats(), []Span{recCall(n, "ok", "backup", 1500, false)}, "ok"},
		{"persistent, chat never", fb, true, okChats(), nil, ""},
	} {
		if got := modelRecovery(t, n, tc.fallbacks, tc.persistent, tc.chats, tc.calls...); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
