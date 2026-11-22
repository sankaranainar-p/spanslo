package core

import "testing"

func tri(v *bool) string {
	if v == nil {
		return "nil"
	}
	if *v {
		return "true"
	}
	return "false"
}

// R-015: baseline, no triggered fault, recovered with pass, recovered with fail, unrecovered.
func TestTaskVerifiedRecovery(t *testing.T) {
	for _, tc := range []struct {
		name      string
		baseline  bool
		recovered []bool
		success   bool
		want      string
	}{
		{"baseline", true, nil, true, "nil"},
		{"baseline fail", true, nil, false, "nil"},
		{"no triggered fault", false, nil, true, "nil"},
		{"all recovered, pass", false, []bool{true, true}, true, "true"},
		{"recovered, fail", false, []bool{true}, false, "false"},
		{"unrecovered, pass", false, []bool{true, false}, true, "false"},
		{"unrecovered, fail", false, []bool{false}, false, "false"},
	} {
		if got := tri(TaskVerifiedRecovery(tc.baseline, tc.recovered, tc.success)); got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
		}
	}
}

// R-026: graceful_error comes from the oracle attribute and is set only with an unrecovered episode.
func TestGracefulError(t *testing.T) {
	n := idxNames(t)
	withFlag := func(v any) []Span {
		ev := successEv(n, 1.0)
		if v != nil {
			ev.Attrs[GracefulErrorAttr] = v
		}
		return []Span{finishedRoot(n, ev)}
	}
	for _, tc := range []struct {
		name      string
		flag      any
		recovered []bool
		want      string
	}{
		{"unrecovered, true", true, []bool{false}, "true"},
		{"unrecovered, false", false, []bool{true, false}, "false"},
		{"unrecovered, absent", nil, []bool{false}, "nil"},
		{"unrecovered, not boolean", "yes", []bool{false}, "nil"},
		{"recovered, true", true, []bool{true}, "nil"},
		{"no episodes, true", true, nil, "nil"},
	} {
		if got := tri(GracefulError(n, withFlag(tc.flag), tc.recovered)); got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
		}
	}
}
