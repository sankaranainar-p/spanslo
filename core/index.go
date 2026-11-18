package core

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// InvalidRun is returned when a run cannot be indexed (R-005, R-016). Reason names the
// offending span, the one with the smallest (start time, lowercase span id).
type InvalidRun struct{ Reason string }

func (e *InvalidRun) Error() string { return e.Reason }

// Call is a tool call span with its step (T, I): T counts from 1, I from 0 within the turn.
type Call struct {
	Span     Span
	Tool     string
	ArgsHash string
	T, I     int
}

// RunIndex is the indexing of one run. Chats[t-1] starts turn t; Calls are in (T, I) order.
type RunIndex struct {
	Chats []Span
	Calls []Call
}

// spanCmp orders by start time, then lowercase span id as a plain string (R-004).
func spanCmp(a, b Span) int {
	if c := a.Start.Compare(b.Start); c != 0 {
		return c
	}
	return strings.Compare(strings.ToLower(a.SpanID), strings.ToLower(b.SpanID))
}

// callArgsHash hashes the call arguments: absent is {}, a string is parsed as JSON, anything
// else is marshaled first. Unparseable arguments are an error, no raw string fallback (R-005).
func callArgsHash(n *Names, s Span) (string, error) {
	v, ok := s.Attrs[n.Attributes.ToolCallArguments]
	if !ok {
		return ArgsHash([]byte("{}"))
	}
	if str, ok := v.(string); ok {
		return ArgsHash([]byte(str))
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return ArgsHash(b)
}

// IndexRun assigns turns to chat spans and (t, i) to tool calls (R-004, R-005, R-006).
// The result does not depend on input order. Invalid runs return *InvalidRun.
func IndexRun(n *Names, spans []Span) (*RunIndex, error) {
	var chats []Span
	var calls []Call
	var bad *Span
	var reason string
	flag := func(s Span, why string) {
		if bad == nil || spanCmp(s, *bad) < 0 {
			bad, reason = &s, fmt.Sprintf("span %s: %s", s.SpanID, why)
		}
	}
	for _, s := range spans {
		op, _ := s.Attrs[n.Attributes.OperationName].(string)
		switch op {
		case n.Operations.ModelCall:
			chats = append(chats, s)
		case n.Operations.ToolCall:
			h, err := callArgsHash(n, s)
			if err != nil {
				flag(s, "tool call arguments are not valid JSON: "+err.Error())
				continue
			}
			tool, _ := s.Attrs[n.Attributes.ToolName].(string)
			calls = append(calls, Call{Span: s, Tool: tool, ArgsHash: h})
		}
	}
	slices.SortFunc(chats, spanCmp)
	kept := calls[:0]
	for _, c := range calls {
		// Turn t = number of chats starting at or before the call; the last turn is open ended.
		c.T = sort.Search(len(chats), func(i int) bool { return chats[i].Start.After(c.Span.Start) })
		if c.T == 0 {
			flag(c.Span, "tool call starts before the first chat span")
			continue
		}
		kept = append(kept, c)
	}
	if bad != nil {
		return nil, &InvalidRun{reason}
	}
	slices.SortFunc(kept, func(a, b Call) int {
		return cmp.Or(cmp.Compare(a.T, b.T), strings.Compare(a.Tool, b.Tool),
			strings.Compare(a.ArgsHash, b.ArgsHash),
			strings.Compare(strings.ToLower(a.Span.SpanID), strings.ToLower(b.Span.SpanID)))
	})
	for i := range kept {
		if i > 0 && kept[i-1].T == kept[i].T {
			kept[i].I = kept[i-1].I + 1
		}
	}
	return &RunIndex{Chats: chats, Calls: kept}, nil
}
