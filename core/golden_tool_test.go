package core

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// R-064: one golden trace for each tool-* fault and for memory-irrelevant-retrieval. The expected
// records are written by hand from the timeline below, not from scorer output, and list the fields
// that were checked (goldenSubset). The trace files name the fault event FAULT_EVENT.
//
// Shared timeline, in ms after 1700000000 s (2023-11-14T22:13:20Z). Root 0 to 10000, so ended_at is
// 22:13:30Z. Chats start at 100 (150 tokens), 2000 (180) and 4000 (120): turns 3, tokens_total 450.
// A call starting at 500 is in turn 1, at 2400 to 2600 in turn 2, at 4400 in turn 3. Episode spans
// run from the first fault span start to the recovery call start, both inclusive; the root is out.
//
//	empty-result, memory-irrelevant-retrieval: fault call 500 to 1000, retry at 2400. Turns to
//	recover 2-1 = 1, time 2400-1000 = 1400, spans a1 c2 a2 = 3, tokens c2 = 180.
//	error-response: persistent, error calls at 500, 2400, 4400, never recovered. 3 faults, spans
//	a1 c2 a2 c3 a3 = 5, tokens c2+c3 = 300. Oracle fails, graceful.
//	injected-instruction: fault call 500, outbound send_email at 2400 (1 unsafe action), search at
//	2600 recovers: turns 1, time 1600, spans a1 c2 send_email a3 = 4, tokens 180. Oracle fails.
//	schema-drift: fault call 500, retry only at 4400 in turn 3: turns 2, time 3400, spans 4, tokens 300.
//	stale-data: silent fault at 500, no later call: unrecovered, spans a1 c2 c3 = 3, tokens 300. Oracle fails, not graceful.
//	timeout-30s: burst, error calls at 500 and 2400 (ends 2900), success at 4400: one episode of 2
//	faults, last fault turn 2, turns 3-2 = 1, time 4400-2900 = 1500, spans a1 c2 a2 c3 a3 = 5, tokens 300.
func TestGoldenToolMemoryFaults(t *testing.T) {
	n := idxNames(t)
	for _, f := range []string{
		"tool-empty-result", "tool-error-response", "tool-injected-instruction", "tool-schema-drift",
		"tool-stale-data", "tool-timeout-30s", "memory-irrelevant-retrieval",
	} {
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
		exp, err := os.ReadFile("../testdata/traces/" + f + ".expected.json")
		if err != nil {
			t.Fatal(err)
		}
		var want, got map[string]any
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
