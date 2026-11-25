package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
)

// goldenSubset fails for each key of want that is missing or different in got, so an
// expected record lists only the fields that were checked by hand.
func goldenSubset(t *testing.T, path string, want, got any) {
	t.Helper()
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			t.Errorf("%s: got %v, want an object", path, got)
			return
		}
		for k, wv := range w {
			gv, ok := g[k]
			if !ok {
				t.Errorf("%s.%s: missing", path, k)
				continue
			}
			goldenSubset(t, path+"."+k, wv, gv)
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			t.Errorf("%s: got %v, want %d items", path, got, len(w))
			return
		}
		for i := range w {
			goldenSubset(t, fmt.Sprintf("%s[%d]", path, i), w[i], g[i])
		}
	default:
		if !reflect.DeepEqual(want, got) {
			t.Errorf("%s: got %v, want %v", path, got, want)
		}
	}
}

// R-064: one golden trace per model-* fault scores to its hand written expected record.
// The trace files name the fault event as FAULT_EVENT; the test swaps in FaultEventName.
func TestGoldenModelFaults(t *testing.T) {
	n := idxNames(t)
	for _, f := range []string{"model-malformed-tool-args", "model-rate-limit-429", "model-server-error-5xx", "model-timeout", "model-truncated-stream"} {
		if _, err := os.Stat("../catalog/" + f + ".yaml"); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile("../testdata/traces/" + f + ".json")
		if err != nil {
			t.Fatal(err)
		}
		spans, err := ParseOTLP(bytes.ReplaceAll(raw, []byte("FAULT_EVENT"), []byte(FaultEventName)))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		runs := GroupRuns(n, spans)
		if len(runs) != 1 {
			t.Fatalf("%s: got %d runs, want 1", f, len(runs))
		}
		var want, got map[string]any
		exp, err := os.ReadFile("../testdata/traces/" + f + ".expected.json")
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(exp, &want); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, run := range runs {
			rec, warns, inv := ScoreRun(n, defaultContract, run)
			if inv != nil || len(warns) != 0 {
				t.Fatalf("%s: invalid %v, warnings %v", f, inv, warns)
			}
			b, err := json.Marshal(rec)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatal(err)
			}
		}
		goldenSubset(t, f, want, got)
	}
}
