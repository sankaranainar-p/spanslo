package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// subset fails for each key of want that is missing or different in got (same rule as core's goldenSubset).
func subset(t *testing.T, path string, want, got any) {
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
			subset(t, path+"."+k, wv, gv)
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			t.Errorf("%s: got %v, want %d items", path, got, len(w))
			return
		}
		for i := range w {
			subset(t, fmt.Sprintf("%s[%d]", path, i), w[i], g[i])
		}
	default:
		if !reflect.DeepEqual(want, got) {
			t.Errorf("%s: got %v, want %v", path, got, want)
		}
	}
}

// R-064: every golden trace, scored by the real score command, matches its expected record.
func TestScoreGoldenTraces(t *testing.T) {
	files, err := filepath.Glob("../../testdata/traces/*.expected.json")
	if err != nil || len(files) != 12 {
		t.Fatalf("got %d fixtures (%v), want 12", len(files), err)
	}
	for _, expPath := range files {
		name := strings.TrimSuffix(filepath.Base(expPath), ".expected.json")
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "runs.jsonl")
			var stdout, stderr bytes.Buffer
			args := []string{"score", "-traces", "../../testdata/traces/" + name + ".json", "-catalog", "../../catalog", "-out", out}
			if code := run(args, &stdout, &stderr); code != 0 {
				t.Fatalf("exit %d: %s%s", code, stdout.String(), stderr.String())
			}
			b, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(string(b)), "\n")
			if len(lines) != 1 || lines[0] == "" {
				t.Fatalf("got %d records, want 1: %s", len(lines), stdout.String())
			}
			var got, want map[string]any
			if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
				t.Fatal(err)
			}
			exp, err := os.ReadFile(expPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(exp, &want); err != nil {
				t.Fatal(err)
			}
			if got["fault_not_applied"] != false {
				t.Errorf("fault_not_applied = %v, want false", got["fault_not_applied"])
			}
			subset(t, name, want, got)
		})
	}
}
