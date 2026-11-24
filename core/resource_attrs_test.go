package core

import (
	"reflect"
	"testing"
)

func strAttr(k, v string) string {
	return `{"key":"` + k + `","value":{"stringValue":"` + v + `"}}`
}

// R-001, R-078: identity written only as resource attributes groups into one run and ScoreRun reads it.
func TestResourceAttrsIdentity(t *testing.T) {
	n := idxNames(t)
	id := n.Spanslo.RunIdentity
	op, er := n.Attributes.OperationName, n.Events.EvaluationResult
	root := `{"traceId":"t1","spanId":"r","name":"root","startTimeUnixNano":"1700000000000000000","endTimeUnixNano":"1700000002000000000",` +
		`"attributes":[` + strAttr(op, n.Operations.Run) + `,{"key":"` + BaselineAttr + `","value":{"boolValue":true}}],` +
		`"events":[{"name":"` + er.Name + `","timeUnixNano":"1700000001000000000","attributes":[` +
		strAttr(er.Attrs.EvaluationName, er.SuccessName) + `,{"key":"` + er.Attrs.ScoreValue + `","value":{"doubleValue":1}}]}]}`
	chat := `{"traceId":"t2","spanId":"c","parentSpanId":"r","name":"chat","startTimeUnixNano":"1700000000500000000","endTimeUnixNano":"1700000000600000000","attributes":[` +
		strAttr(op, n.Operations.ModelCall) + `]}`
	res := strAttr(id.RunID, "run-1") + "," + strAttr(id.ExperimentID, "exp-1") + "," +
		strAttr(id.Framework, "fw") + "," + strAttr(id.Model, "m1")
	data := `{"resourceSpans":[{"resource":{"attributes":[` + res + `]},"scopeSpans":[{"spans":[` + root + "," + chat + `]}]}]}`
	spans, err := ParseOTLP([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if runs := GroupRuns(n, spans); len(runs) != 1 || len(runs["run-1"]) != 2 {
		t.Fatalf("got %d runs, want one run-1 with 2 spans", len(runs))
	}
	rec, _, inv := ScoreRun(n, defaultContract, spans)
	if inv != nil {
		t.Fatal(inv.Reason)
	}
	if rec.RunID != "run-1" || rec.ExperimentID != "exp-1" || rec.Framework != "fw" || rec.Model != "m1" {
		t.Errorf("got %+v", rec)
	}
}

// R-001, R-078: a span's own attribute beats the resource's; no resource attributes leaves the span as is.
func TestResourceAttrsSpanWins(t *testing.T) {
	data := `{"resourceSpans":[` +
		`{"resource":{"attributes":[` + strAttr("k", "res") + `,` + strAttr("other", "r2") + `]},` +
		`"scopeSpans":[{"spans":[{"spanId":"a","attributes":[` + strAttr("k", "span") + `]}]}]},` +
		`{"resource":{},"scopeSpans":[{"spans":[{"spanId":"b","attributes":[` + strAttr("k", "own") + `]},{"spanId":"c"}]}]}]}`
	spans, err := ParseOTLP([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if want := (map[string]any{"k": "span", "other": "r2"}); !reflect.DeepEqual(spans[0].Attrs, want) {
		t.Errorf("a: got %v", spans[0].Attrs)
	}
	if want := (map[string]any{"k": "own"}); !reflect.DeepEqual(spans[1].Attrs, want) {
		t.Errorf("b: got %v", spans[1].Attrs)
	}
	if len(spans[2].Attrs) != 0 {
		t.Errorf("c: got %v", spans[2].Attrs)
	}
}
