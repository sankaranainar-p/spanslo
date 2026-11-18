package core

import (
	"errors"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func idxNames(t *testing.T) *Names {
	t.Helper()
	data, err := os.ReadFile("../spec/semconv-mapping.yaml")
	if err != nil {
		t.Fatal(err)
	}
	n, err := LoadNames(data)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func mkSpan(n *Names, id, op string, ms int, attrs map[string]any) Span {
	attrs[n.Attributes.OperationName] = op
	return Span{SpanID: id, Start: time.Unix(1700000000, 0).Add(time.Duration(ms) * time.Millisecond), Attrs: attrs}
}

func mkChat(n *Names, id string, ms int) Span {
	return mkSpan(n, id, n.Operations.ModelCall, ms, map[string]any{})
}

// mkCall builds a tool call; nil args means the arguments attribute is absent.
func mkCall(n *Names, id, tool string, ms int, args any) Span {
	a := map[string]any{n.Attributes.ToolName: tool}
	if args != nil {
		a[n.Attributes.ToolCallArguments] = args
	}
	return mkSpan(n, id, n.Operations.ToolCall, ms, a)
}

func stepsOf(idx *RunIndex) map[string][2]int {
	m := map[string][2]int{}
	for _, c := range idx.Calls {
		m[c.Span.SpanID] = [2]int{c.T, c.I}
	}
	return m
}

func mustIndex(t *testing.T, n *Names, spans []Span) *RunIndex {
	t.Helper()
	idx, err := IndexRun(n, spans)
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

// R-004, R-005, R-006: same (t, i) for any input order, equal start times (chats and
// parallel calls with the same tool and arguments), and mixed case span ids.
func TestIndexRunOrderIndependent(t *testing.T) {
	n := idxNames(t)
	same := `{"q":"x"}`
	spans := []Span{
		mkChat(n, "c", 1000), mkChat(n, "B", 0), mkChat(n, "a", 0),
		mkCall(n, "z9", "search", 500, same),
		mkCall(n, "Y1", "search", 500, same),
		mkCall(n, "al", "alpha", 500, `{"q":"y"}`),
		mkCall(n, "k1", "kk", 1000, nil),
		mkCall(n, "l1", "ll", 9000, nil),
	}
	want := map[string][2]int{
		"al": {2, 0}, "Y1": {2, 1}, "z9": {2, 2}, "k1": {3, 0}, "l1": {3, 1},
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		rng.Shuffle(len(spans), func(a, b int) { spans[a], spans[b] = spans[b], spans[a] })
		idx := mustIndex(t, n, spans)
		if got := stepsOf(idx); !reflect.DeepEqual(got, want) {
			t.Fatalf("shuffle %d: got %v, want %v", i, got, want)
		}
		var ids []string
		for _, c := range idx.Chats {
			ids = append(ids, c.SpanID)
		}
		if !reflect.DeepEqual(ids, []string{"a", "B", "c"}) {
			t.Fatalf("chat order %v", ids)
		}
	}
}

// R-005: a call at the same instant as chat t+1 belongs to t+1; the last turn is open ended.
func TestIndexRunTurnBoundaries(t *testing.T) {
	n := idxNames(t)
	idx := mustIndex(t, n, []Span{
		mkChat(n, "c1", 0), mkChat(n, "c2", 1000),
		mkCall(n, "a", "t", 999, nil), mkCall(n, "b", "t", 1000, nil), mkCall(n, "c", "t", 3600000, nil),
	})
	want := map[string][2]int{"a": {1, 0}, "b": {2, 0}, "c": {2, 1}}
	if got := stepsOf(idx); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// R-005: a call before the first chat span, or with unparseable arguments, makes the run
// invalid; the reason names the span with the smallest (start time, lowercase span id).
func TestIndexRunInvalid(t *testing.T) {
	n := idxNames(t)
	chat := mkChat(n, "c1", 100)
	for _, tc := range []struct {
		name, want string
		spans      []Span
	}{
		{"same start, lowercase id", "AA", []Span{chat, mkCall(n, "BB", "t", 50, nil), mkCall(n, "AA", "t", 50, nil)}},
		{"earlier bad arguments", "zz", []Span{chat, mkCall(n, "AA", "t", 50, nil), mkCall(n, "zz", "t", 10, "{oops")}},
		{"later bad arguments", "AA", []Span{chat, mkCall(n, "AA", "t", 50, nil), mkCall(n, "zz", "t", 150, "{oops")}},
		{"bad arguments only", "bad", []Span{chat, mkCall(n, "bad", "t", 150, "not json")}},
		{"no chat spans, no turn 0", "only", []Span{mkCall(n, "only", "t", 0, nil)}},
	} {
		_, err := IndexRun(n, tc.spans)
		var inv *InvalidRun
		if !errors.As(err, &inv) || !strings.Contains(inv.Reason, "span "+tc.want+":") {
			t.Errorf("%s: got %v, want invalid run naming span %s", tc.name, err, tc.want)
		}
	}
}

// R-005: string and structured arguments with equal JSON hash alike; absent equals "{}".
func TestIndexRunArgsHash(t *testing.T) {
	n := idxNames(t)
	hash := func(args any) string {
		return mustIndex(t, n, []Span{mkChat(n, "c", 0), mkCall(n, "x", "t", 1, args)}).Calls[0].ArgsHash
	}
	empty, _ := ArgsHash([]byte("{}"))
	if h := hash(nil); h != empty {
		t.Errorf("absent: %s, want %s", h, empty)
	}
	if h := hash("{}"); h != empty {
		t.Errorf(`"{}": %s, want %s`, h, empty)
	}
	s := hash(`{ "b": 1, "a": [2] }`)
	st := hash(map[string]any{"a": []any{int64(2)}, "b": int64(1)})
	if s != st || s == empty {
		t.Errorf("string %s, structured %s", s, st)
	}
}
