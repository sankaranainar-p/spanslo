package core

// Episode is one run of triggered faults on one target with no recovery between them
// (R-010, R-011). Recovery fields are filled by later tasks.
type Episode struct {
	Target                        string
	FirstFaultTurn, LastFaultTurn int64
	FaultCount                    int
}

// GroupEpisodes groups triggered faults into episodes per target, in order of first fault
// per target (R-010). Untriggered faults are skipped: they never start or join an episode
// (R-028). A fault starts a new episode when a qualifying recovery (R-012, R-013) lies
// between the previous fault of that target and it; the recovery logic is FindRecoveryFor's.
// faults must be in time order, as TriggeredFaults returns them.
func GroupEpisodes(idx *RunIndex, spans []Span, faults []FaultEvent, fallbacks []string, persistent bool) []Episode {
	byID := make(map[string]Span, len(spans))
	for _, s := range spans {
		byID[s.SpanID] = s
	}
	var order []string
	byTarget := map[string][]FaultEvent{}
	for _, f := range faults {
		if !f.Triggered {
			continue
		}
		if _, ok := byTarget[f.Target]; !ok {
			order = append(order, f.Target)
		}
		byTarget[f.Target] = append(byTarget[f.Target], f)
	}
	var out []Episode
	for _, target := range order {
		var prev FaultEvent
		for i, f := range byTarget[target] {
			if i == 0 || recoveredBetween(idx, byID[prev.SpanID], byID[f.SpanID], prev.Surface, target, fallbacks, persistent) {
				out = append(out, Episode{Target: target, FirstFaultTurn: f.Turn})
			}
			e := &out[len(out)-1]
			e.LastFaultTurn = f.Turn
			e.FaultCount++
			prev = f
		}
	}
	return out
}

// recoveredBetween reports whether FindRecoveryFor finds a qualifying recovery for fault span a
// among the spans that start before fault span b. Cutting the spans at b keeps b's own error
// from cancelling the recovery; "no later error" (R-012) is judged inside that window only.
func recoveredBetween(idx *RunIndex, a, b Span, surface, target string, fallbacks []string, persistent bool) bool {
	return FindRecoveryFor(cutBefore(idx, b), a, surface, target, fallbacks, persistent) != nil
}
