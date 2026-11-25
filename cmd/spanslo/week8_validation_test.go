package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sankaranainar-p/spanslo/core"
)

// R-065: week 8 validation, end to end through the dispatcher and the files score writes.
// score deletes a stale sidecar and rewrites it; gate exits 1 on a breaching run file, and 2
// when the sidecar is missing and when it is non empty (invalid run); score writes no sidecar
// when it exits 2.
func TestWeek8Validation(t *testing.T) {
	dir, n := setup(t)
	out := filepath.Join(dir, "runs.jsonl")
	sidecar := out + ".invalid.jsonl"
	sloFile := filepath.Join(dir, "gate.yaml")
	write(t, sloFile, okSLO)
	gateExit := func() int {
		var buf bytes.Buffer
		return run([]string{"gate", "--runs", out, "--slo", sloFile}, &buf, &buf)
	}
	size := func() int64 {
		fi, err := os.Stat(sidecar)
		if err != nil {
			t.Fatal(err)
		}
		return fi.Size()
	}

	// a stale sidecar is replaced by an empty one when all runs are valid
	write(t, sidecar, "stale\n")
	if code, so, se := runScore(t, dir, jsonl(t, traceFor(n, "b1", true, true))); code != 0 || size() != 0 {
		t.Fatalf("clean score: exit %d, sidecar size %d, stdout %q, stderr %q", code, size(), so, se)
	}

	// gate exits 1 on a breaching run file next to score's empty sidecar
	var sb bytes.Buffer
	for i := 1; i <= 2; i++ {
		b, _ := json.Marshal(sloRec(fmt.Sprintf("r%d", i), i, func(r *core.RunRecord) { r.UnsafeActions = 1 }))
		sb.Write(append(b, '\n'))
	}
	write(t, out, sb.String())
	if got := gateExit(); got != 1 {
		t.Errorf("breach: gate exit %d, want 1", got)
	}

	// gate exits 2 when the sidecar is missing
	if err := os.Remove(sidecar); err != nil {
		t.Fatal(err)
	}
	if got := gateExit(); got != 2 {
		t.Errorf("missing sidecar: gate exit %d, want 2", got)
	}

	// an invalid run (no task success event) makes score write a non empty sidecar; gate exits 2
	if code, so, se := runScore(t, dir, jsonl(t, traceFor(n, "bad", true, false))); code != 0 || size() == 0 {
		t.Fatalf("invalid run: score exit %d, sidecar size %d, stdout %q, stderr %q", code, size(), so, se)
	}
	if got := gateExit(); got != 2 {
		t.Errorf("non empty sidecar: gate exit %d, want 2", got)
	}

	// score exits 2 on unparseable traces and leaves no sidecar, stale or new
	write(t, sidecar, "stale\n")
	if code, _, _ := runScore(t, dir, "not json"); code != 2 {
		t.Errorf("bad traces: score exit %d, want 2", code)
	}
	if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
		t.Errorf("bad traces: sidecar still there (stat err %v)", err)
	}
}
