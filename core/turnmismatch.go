package core

// FaultEventName and FaultTurnAttr name the canonical fault event and its turn attribute
// (R-057). Like RunIDAttr they are spanslo names, not gen_ai names.
const (
	FaultEventName = "spanslo.fault"
	FaultTurnAttr  = "spanslo.fault.turn"
)

// TurnMismatches counts fault events on indexed spans (chat and tool call) whose turn differs
// from the turn derived for that span; missing or non integer turns count (R-007).
func TurnMismatches(idx *RunIndex, spans []Span) int {
	turn := map[string]int{}
	for i, c := range idx.Chats {
		turn[c.SpanID] = i + 1
	}
	for _, c := range idx.Calls {
		turn[c.Span.SpanID] = c.T
	}
	n := 0
	for _, s := range spans {
		t, ok := turn[s.SpanID]
		if !ok {
			continue
		}
		for _, e := range s.Events {
			if e.Name != FaultEventName {
				continue
			}
			if v, isInt := e.Attrs[FaultTurnAttr].(int64); !isInt || v != int64(t) {
				n++
			}
		}
	}
	return n
}
