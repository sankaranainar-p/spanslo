package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sankaranainar-p/spanslo/core"
)

// R-068: exit code audit through the dispatcher. An unknown flag, a missing file, unparseable
// runs, traces or SLO file and an unparseable ended_at all exit 2; a breach exits 0 for slo and
// report and 1 for gate. score has no runs, SLO file or ended_at input, so those rows are absent.
func TestExitCodeAudit(t *testing.T) {
	dir, n := setup(t)
	put := func(name, s string) string {
		p := filepath.Join(dir, name)
		write(t, p, s)
		return p
	}
	// recs writes two triggered run records and the empty invalid run sidecar that gate wants.
	recs := func(name string, mut func(*core.RunRecord)) string {
		var sb strings.Builder
		for i := 1; i <= 2; i++ {
			b, _ := json.Marshal(sloRec(fmt.Sprintf("r%d", i), i, mut))
			sb.Write(append(b, '\n'))
		}
		put(name+".invalid.jsonl", "")
		return put(name, sb.String())
	}
	okRuns := recs("ok.jsonl", nil)
	breachRuns := recs("breach.jsonl", func(r *core.RunRecord) { r.UnsafeActions = 1 })
	dateRuns := recs("date.jsonl", func(r *core.RunRecord) { r.EndedAt = "2026-02-30T00:00:00Z" })
	put("bad.jsonl.invalid.jsonl", "")
	badRuns := put("bad.jsonl", "{not json\n")
	sloFile, badSLO := put("slo.yaml", okSLO), put("bad.yaml", "slos: [")
	missing := filepath.Join(dir, "missing")
	catalog := filepath.Join(dir, "catalog")
	traces := put("traces.jsonl", jsonl(t, traceFor(n, "b1", true, true)))
	badTraces := put("bad-traces.jsonl", "not json")
	out := filepath.Join(dir, "out.jsonl")

	type tc struct {
		name string
		args []string
		want int
	}
	var cases []tc
	for _, cmd := range []string{"slo", "report", "gate"} {
		args := func(runs, sl string) []string {
			a := []string{cmd, "--runs", runs, "--slo", sl}
			if cmd == "report" {
				a = append(a, "--out", out)
			}
			return a
		}
		breach := 0
		if cmd == "gate" {
			breach = 1
		}
		cases = append(cases,
			tc{cmd + " unknown flag", []string{cmd, "--bogus"}, 2},
			tc{cmd + " missing runs file", args(missing, sloFile), 2},
			tc{cmd + " missing slo file", args(okRuns, missing), 2},
			tc{cmd + " unparseable runs", args(badRuns, sloFile), 2},
			tc{cmd + " unparseable slo file", args(okRuns, badSLO), 2},
			tc{cmd + " unparseable ended_at", args(dateRuns, sloFile), 2},
			tc{cmd + " breach", args(breachRuns, sloFile), breach},
		)
	}
	cases = append(cases,
		tc{"score unknown flag", []string{"score", "--bogus"}, 2},
		tc{"score missing traces file", []string{"score", "--traces", missing, "--catalog", catalog, "--out", out}, 2},
		tc{"score missing catalog", []string{"score", "--traces", traces, "--catalog", missing, "--out", out}, 2},
		tc{"score unparseable traces", []string{"score", "--traces", badTraces, "--catalog", catalog, "--out", out}, 2},
	)
	for _, c := range cases {
		var buf bytes.Buffer
		if got := run(c.args, &buf, &buf); got != c.want {
			t.Errorf("%s: exit %d, want %d; output %q", c.name, got, c.want, buf.String())
		}
	}
}
