package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sankaranainar-p/spanslo/core"
)

type kv = map[string]any

func attr(k string, v any) kv {
	switch x := v.(type) {
	case bool:
		return kv{"key": k, "value": kv{"boolValue": x}}
	case float64:
		return kv{"key": k, "value": kv{"doubleValue": x}}
	}
	return kv{"key": k, "value": kv{"stringValue": v}}
}

const baseMs = 1_700_000_000_000

func span(run, id string, ms, endMs int64, attrs, events []kv) kv {
	return kv{"traceId": "t-" + run, "spanId": run + id, "name": "s",
		"startTimeUnixNano": strconv.FormatInt((baseMs+ms)*1_000_000, 10),
		"endTimeUnixNano":   strconv.FormatInt((baseMs+endMs)*1_000_000, 10),
		"attributes": attrs, "events": events}
}

// resFor is the resource attributes of one run: the identity the harness writes.
func resFor(n *core.Names, run string) []kv {
	ri := n.Spanslo.RunIdentity
	return []kv{attr(ri.RunID, run), attr(ri.ExperimentID, "e"), attr(ri.Framework, "fw"), attr(ri.Model, "m")}
}

// wrap puts spans under one resource.
func wrap(res []kv, spans ...kv) kv {
	return kv{"resourceSpans": []kv{{"resource": kv{"attributes": res}, "scopeSpans": []kv{{"spans": spans}}}}}
}

// successEv is the passing task success event.
func successEv(n *core.Names) kv {
	er := n.Events.EvaluationResult
	return kv{"name": er.Name, "timeUnixNano": "1", "attributes": []kv{
		attr(er.Attrs.EvaluationName, er.SuccessName), attr(er.Attrs.ScoreValue, 1.0)}}
}

// traceFor is one run: identity as resource attributes, a root, one model call and two calls
// to one tool with different arguments. success false leaves out the task success event.
func traceFor(n *core.Names, run string, baseline, success bool, faults ...string) kv {
	op := n.Attributes.OperationName
	root := []kv{attr(op, n.Operations.Run)}
	if baseline {
		root = append(root, attr(core.BaselineAttr, true))
	} else {
		root = append(root, attr(core.ModeAttr, "single"))
	}
	var evs []kv
	if success {
		evs = append(evs, successEv(n))
	}
	for _, f := range faults { // untriggered, so they need no indexed span
		evs = append(evs, kv{"name": core.FaultEventName, "timeUnixNano": "1", "attributes": []kv{
			attr("spanslo.fault.id", f), attr("spanslo.fault.target", "search"),
			{"key": core.FaultTurnAttr, "value": kv{"intValue": "1"}}, attr("spanslo.fault.triggered", false)}})
	}
	call := func(id string, ms int64, args string) kv {
		return span(run, id, ms, ms+50, []kv{attr(op, n.Operations.ToolCall),
			attr(n.Attributes.ToolName, "t"), attr(n.Attributes.ToolCallArguments, args)}, nil)
	}
	return wrap(resFor(n, run),
		span(run, "r", 0, 5000, root, evs),
		span(run, "c", 1000, 1100, []kv{attr(op, n.Operations.ModelCall)}, nil),
		call("1", 1200, `{"i":1}`), call("2", 1300, `{"i":2}`))
}

// warnTrace is a triggered model surface fault on chat c1, which ends at 6000 ms. The recovering
// chat c2 starts at 2000 ms, before that, so time_to_recover_ms clamps to 0 and core warns (R-018).
func warnTrace(n *core.Names, run string) kv {
	op := n.Attributes.OperationName
	ev := kv{"name": core.FaultEventName, "timeUnixNano": "1", "attributes": []kv{
		attr("spanslo.fault.id", "strict"), attr("spanslo.fault.target", "gpt"),
		attr("spanslo.fault.surface", "model"),
		{"key": core.FaultTurnAttr, "value": kv{"intValue": "1"}}, attr("spanslo.fault.triggered", true)}}
	c1 := span(run, "c1", 1000, 6000, []kv{attr(op, n.Operations.ModelCall)}, []kv{ev})
	c1["status"] = kv{"code": 2}
	return wrap(resFor(n, run),
		span(run, "r", 0, 8000, []kv{attr(op, n.Operations.Run), attr(core.ModeAttr, "single")}, []kv{successEv(n)}),
		c1, span(run, "c2", 2000, 2100, []kv{attr(op, n.Operations.ModelCall)}, nil))
}

// setup copies the spec names file and the default contract into a temp tree, next to a fault
// file "strict" that allows no retries.
func setup(t *testing.T) (string, *core.Names) {
	t.Helper()
	dir := t.TempDir()
	for src, dst := range map[string]string{
		"../../catalog/defaults.yaml":     "catalog/defaults.yaml",
		"../../spec/semconv-mapping.yaml": "spec/semconv-mapping.yaml",
	} {
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(dir, dst), string(b))
	}
	write(t, filepath.Join(dir, "catalog", "strict.yaml"),
		"contract:\n  efficiency:\n    max_retries_same_call: 0\n    max_retries_same_tool: 0\n")
	b, err := os.ReadFile(filepath.Join(dir, "spec", "semconv-mapping.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	n, err := core.LoadNames(b)
	if err != nil {
		t.Fatal(err)
	}
	return dir, n
}

func write(t *testing.T, path, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func jsonl(t *testing.T, ts ...kv) string {
	t.Helper()
	var sb strings.Builder
	for _, x := range ts {
		b, err := json.Marshal(x)
		if err != nil {
			t.Fatal(err)
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// runScore writes the traces and runs score; out is <dir>/runs.jsonl. It returns code, stdout, stderr.
func runScore(t *testing.T, dir, traces string) (int, string, string) {
	t.Helper()
	p := filepath.Join(dir, "traces.jsonl")
	write(t, p, traces)
	var stdout, stderr bytes.Buffer
	code := score([]string{"--traces", p, "--catalog", filepath.Join(dir, "catalog"),
		"--out", filepath.Join(dir, "runs.jsonl")}, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func readLines(t *testing.T, path string) []kv {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []kv
	for _, l := range strings.Split(string(b), "\n") {
		if l == "" {
			continue
		}
		var m kv
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
	}
	return out
}

// R-060, R-016, R-025: mixed input exits 0 and names the invalid runs in the sidecar; a baseline
// run is scored with the default contract, a faulted run with the contract of its fault id, and
// two distinct fault ids make the run invalid.
func TestScoreMixedInput(t *testing.T) {
	dir, n := setup(t)
	code, stdout, _ := runScore(t, dir, jsonl(t,
		traceFor(n, "b1", true, true),
		traceFor(n, "bad", true, false),
		traceFor(n, "f1", false, true, "strict"),
		traceFor(n, "f2", false, true, "strict", "other"),
		traceFor(n, "f3", false, true, "nosuch"),
	))
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	recs := readLines(t, filepath.Join(dir, "runs.jsonl"))
	if len(recs) != 2 || recs[0]["run_id"] != "b1" || recs[1]["run_id"] != "f1" {
		t.Fatalf("records: %v", recs)
	}
	if recs[0]["retry_budget_violations"] != 0.0 || recs[1]["retry_budget_violations"] != 1.0 {
		t.Errorf("baseline want default contract (0 violations), faulted want strict (1): %v", recs)
	}
	bad := readLines(t, filepath.Join(dir, "runs.jsonl.invalid.jsonl"))
	if len(bad) != 3 || bad[0]["run_id"] != "bad" || bad[1]["run_id"] != "f2" || bad[2]["run_id"] != "f3" {
		t.Fatalf("sidecar: %v", bad)
	}
	if r, _ := bad[1]["reason"].(string); !strings.Contains(r, "spanslo.fault.id") {
		t.Errorf("f2 reason %q does not name spanslo.fault.id", r)
	}
	if !strings.Contains(stdout, "run f2: invalid") || !strings.Contains(stdout, "run b1: ok") {
		t.Errorf("stdout: %s", stdout)
	}
}

// R-060: a stale sidecar is replaced, here by an empty one.
func TestScoreStaleSidecarReplaced(t *testing.T) {
	dir, n := setup(t)
	side := filepath.Join(dir, "runs.jsonl.invalid.jsonl")
	write(t, side, `{"run_id":"stale","reason":"old"}`+"\n")
	code, _, _ := runScore(t, dir, jsonl(t, traceFor(n, "b1", true, true)))
	if b, err := os.ReadFile(side); code != 0 || err != nil || len(b) != 0 {
		t.Errorf("exit %d, err %v, sidecar %q, want 0, nil, empty", code, err, b)
	}
}

// R-060: unparseable input and a usage error exit 2 and leave no sidecar, stale or new.
func TestScoreExit2NoSidecar(t *testing.T) {
	dir, _ := setup(t)
	side := filepath.Join(dir, "runs.jsonl.invalid.jsonl")
	write(t, side, "stale\n")
	if code, _, _ := runScore(t, dir, "not json"); code != 2 {
		t.Errorf("unparseable input: exit %d, want 2", code)
	}
	if _, err := os.Stat(side); !os.IsNotExist(err) {
		t.Errorf("unparseable input left a sidecar (%v)", err)
	}
	write(t, side, "stale\n")
	var buf bytes.Buffer
	if code := score([]string{"--out", filepath.Join(dir, "runs.jsonl")}, &buf, &buf); code != 2 {
		t.Errorf("usage error: exit %d, want 2", code)
	}
	if _, err := os.Stat(side); !os.IsNotExist(err) {
		t.Errorf("usage error left a sidecar (%v)", err)
	}
}

// R-018: a core warning (clamped time_to_recover_ms) goes to stderr as "warning: run <id>:";
// the run still scores and the command exits 0.
func TestScoreWarningsToStderr(t *testing.T) {
	dir, n := setup(t)
	code, _, stderr := runScore(t, dir, jsonl(t, warnTrace(n, "w1")))
	if code != 0 || !strings.Contains(stderr, "warning: run w1:") {
		t.Errorf("exit %d, stderr %q, want 0 and a warning for run w1", code, stderr)
	}
	recs := readLines(t, filepath.Join(dir, "runs.jsonl"))
	if len(recs) != 1 || recs[0]["run_id"] != "w1" {
		t.Errorf("records: %v", recs)
	}
	if bad := readLines(t, filepath.Join(dir, "runs.jsonl.invalid.jsonl")); len(bad) != 0 {
		t.Errorf("sidecar: %v", bad)
	}
}

// R-025: a fault id that is not a plain catalog file name is an invalid run, never a path read.
func TestScoreFaultIDNotAFileName(t *testing.T) {
	dir, n := setup(t)
	code, _, _ := runScore(t, dir, jsonl(t, traceFor(n, "p1", false, true, "../x")))
	bad := readLines(t, filepath.Join(dir, "runs.jsonl.invalid.jsonl"))
	if code != 0 || len(bad) != 1 {
		t.Fatalf("exit %d, sidecar %v, want 0 and one invalid run", code, bad)
	}
	if r, _ := bad[0]["reason"].(string); !strings.Contains(r, "catalog file name") {
		t.Errorf("reason %q", r)
	}
}

// R-056, R-025: a bench fault record (the bench adapter's short field names, source bench) is
// converted before scoring, so its id picks the strict contract. Without the adapter step the
// record has no spanslo.fault.* names, the run is invalid and no record is written.
func TestScoreConvertsBenchRecords(t *testing.T) {
	dir, n := setup(t)
	tr := traceFor(n, "a1", false, true)
	root := tr["resourceSpans"].([]kv)[0]["scopeSpans"].([]kv)[0]["spans"].([]kv)[0]
	root["events"] = append(root["events"].([]kv), kv{"name": core.FaultEventName, "timeUnixNano": "1", "attributes": []kv{
		attr("source", "bench"), attr("id", "strict"), attr("surface", "tool"), attr("target", "search"),
		{"key": "turn", "value": kv{"intValue": "1"}}, attr("triggered", false)}})
	code, _, _ := runScore(t, dir, jsonl(t, tr))
	recs := readLines(t, filepath.Join(dir, "runs.jsonl"))
	if code != 0 || len(recs) != 1 || recs[0]["retry_budget_violations"] != 1.0 {
		t.Fatalf("exit %d, records %v, want 0 and one record scored with the strict contract", code, recs)
	}
}

// R-068: no subcommand and an unknown subcommand print usage and exit 2.
func TestRunDispatch(t *testing.T) {
	var out, errb bytes.Buffer
	for _, args := range [][]string{nil, {"bogus"}} {
		if code := run(args, &out, &errb); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	if !strings.Contains(errb.String(), "usage") {
		t.Errorf("stderr %q has no usage", errb.String())
	}
}
