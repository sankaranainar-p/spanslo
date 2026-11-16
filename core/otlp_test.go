package core

import (
	"reflect"
	"testing"
	"time"
)

// Two exporter lines: a finished span with every value type and an event, then an
// unfinished span with a string status code.
const otlpFixture = `{"resourceSpans":[{"scopeSpans":[{"spans":[{"traceId":"0102030405060708090a0b0c0d0e0f10","spanId":"0102030405060708","name":"root","startTimeUnixNano":"1700000000000000000","endTimeUnixNano":"1700000002500000000","attributes":[{"key":"s","value":{"stringValue":"v"}},{"key":"i","value":{"intValue":"42"}},{"key":"n","value":{"intValue":7}},{"key":"d","value":{"doubleValue":1.5}},{"key":"b","value":{"boolValue":true}},{"key":"a","value":{"arrayValue":{"values":[{"stringValue":"x"},{"intValue":"1"}]}}},{"key":"m","value":{"kvlistValue":{"values":[{"key":"k","value":{"boolValue":false}}]}}}],"events":[{"timeUnixNano":"1700000001000000000","name":"ev","attributes":[{"key":"e","value":{"stringValue":"1"}}]}],"status":{"code":1,"message":"fine"}}]}]}]}
{"resourceSpans":[{"scopeSpans":[{"spans":[{"traceId":"0102030405060708090a0b0c0d0e0f10","spanId":"1111111111111111","parentSpanId":"0102030405060708","name":"child","startTimeUnixNano":1700000001000000000,"status":{"code":"STATUS_CODE_ERROR"}}]}]}]}`

func TestParseOTLP(t *testing.T) {
	spans, err := ParseOTLP([]byte(otlpFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 2 {
		t.Fatalf("got %d spans, want 2", len(spans))
	}
	root := spans[0]
	wantAttrs := map[string]any{
		"s": "v", "i": int64(42), "n": int64(7), "d": 1.5, "b": true,
		"a": []any{"x", int64(1)}, "m": map[string]any{"k": false},
	}
	if root.Name != "root" || root.SpanID != "0102030405060708" || root.ParentSpanID != "" ||
		root.TraceID != "0102030405060708090a0b0c0d0e0f10" {
		t.Errorf("root ids/name: %+v", root)
	}
	if !reflect.DeepEqual(root.Attrs, wantAttrs) {
		t.Errorf("attrs: got %#v", root.Attrs)
	}
	if !root.Start.Equal(time.Unix(1700000000, 0)) || !root.End.Equal(time.Unix(1700000002, 500000000)) {
		t.Errorf("times: %v %v", root.Start, root.End)
	}
	if len(root.Events) != 1 || root.Events[0].Name != "ev" ||
		!root.Events[0].Time.Equal(time.Unix(1700000001, 0)) ||
		!reflect.DeepEqual(root.Events[0].Attrs, map[string]any{"e": "1"}) {
		t.Errorf("events: %+v", root.Events)
	}
	if root.Status != (Status{StatusOK, "fine"}) {
		t.Errorf("status: %+v", root.Status)
	}
	child := spans[1]
	if child.ParentSpanID != "0102030405060708" || !child.End.IsZero() ||
		!child.Start.Equal(time.Unix(1700000001, 0)) || child.Status.Code != StatusError {
		t.Errorf("child: %+v", child)
	}
}

// JSON null for time and status fields is the same as absent: zero time, unset status.
func TestParseOTLPNullFields(t *testing.T) {
	in := `{"resourceSpans":[{"scopeSpans":[{"spans":[{"name":"n","startTimeUnixNano":null,"endTimeUnixNano":null,"status":{"code":null}}]}]}]}`
	spans, err := ParseOTLP([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 1 || !spans[0].Start.IsZero() || !spans[0].End.IsZero() || spans[0].Status.Code != StatusUnset {
		t.Errorf("got %+v", spans)
	}
}

func TestParseOTLPRejectsBadInput(t *testing.T) {
	for _, in := range []string{"", "{", "not json", `{"resourceSpans":[{"scopeSpans":[{"spans":[{"startTimeUnixNano":"abc"}]}]}]}`} {
		if _, err := ParseOTLP([]byte(in)); err == nil {
			t.Errorf("%q: want error", in)
		}
	}
}
