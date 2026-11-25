package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/sankaranainar-p/spanslo/core"
	"github.com/sankaranainar-p/spanslo/core/adapters"
)

const usage = "usage: spanslo score --traces <file> --catalog <dir> --out <file>"

// contractFor picks the contract of one run (R-025): the default for a baseline run or a run
// with no fault id, the effective contract of the fault id for a faulted run. Faults that carry
// more than one distinct fault id make the run invalid.
func contractFor(catalog string, defaults []byte, def *core.Contract, spans []core.Span) (*core.Contract, *core.InvalidRun) {
	info, err := core.ExtractFaults(spans)
	if err != nil {
		return def, nil // ScoreRun finds the same problem and reports it
	}
	var ids []string
	for _, f := range info.Faults {
		if f.ID != "" && !slices.Contains(ids, f.ID) {
			ids = append(ids, f.ID)
		}
	}
	slices.Sort(ids)
	if len(ids) > 1 {
		return nil, &core.InvalidRun{Reason: fmt.Sprintf("spanslo.fault.id: faults carry %d distinct ids: %s", len(ids), strings.Join(ids, ", "))}
	}
	if info.Baseline || len(ids) == 0 {
		return def, nil
	}
	id := ids[0]
	if !filepath.IsLocal(id) || id != filepath.Base(id) {
		return nil, &core.InvalidRun{Reason: fmt.Sprintf("spanslo.fault.id %q is not a catalog file name", id)}
	}
	fault, err := os.ReadFile(filepath.Join(catalog, id+".yaml"))
	if err != nil {
		return nil, &core.InvalidRun{Reason: fmt.Sprintf("spanslo.fault.id %q: %v", id, err)}
	}
	c, err := core.EffectiveContract(defaults, fault)
	if err != nil {
		return nil, &core.InvalidRun{Reason: fmt.Sprintf("spanslo.fault.id %q: %v", id, err)}
	}
	return c, nil
}

// score implements `spanslo score` (R-060, R-016): exit 0 when input was processed, even with
// invalid runs; exit 2 with no sidecar on a usage error or bad input to the whole command.
func score(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("score", flag.ContinueOnError)
	fs.SetOutput(stderr)
	traces := fs.String("traces", "", "raw OTLP JSON traces file")
	catalog := fs.String("catalog", "", "fault catalog directory")
	out := fs.String("out", "", "run records output file (JSON lines)")
	if err := fs.Parse(args); err != nil {
		if *out != "" { // same stale sidecar rule as the other exit 2 paths
			os.Remove(*out + ".invalid.jsonl")
		}
		return 2
	}
	sidecar := *out + ".invalid.jsonl"
	fail := func(err error) int {
		fmt.Fprintln(stderr, "spanslo score:", err)
		os.Remove(sidecar)
		return 2
	}
	if *out != "" {
		// a stale sidecar must never outlive a run, so it goes before anything can fail
		if err := os.Remove(sidecar); err != nil && !os.IsNotExist(err) {
			return fail(err)
		}
	}
	if *traces == "" || *catalog == "" || *out == "" || fs.NArg() > 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	// ponytail: the names file is found next to the catalog (<catalog>/../spec); add a flag if layouts differ.
	semconv, err := os.ReadFile(filepath.Join(*catalog, "..", "spec", "semconv-mapping.yaml"))
	if err != nil {
		return fail(err)
	}
	n, err := core.LoadNames(semconv)
	if err != nil {
		return fail(err)
	}
	defaults, err := os.ReadFile(filepath.Join(*catalog, "defaults.yaml"))
	if err != nil {
		return fail(err)
	}
	def, err := core.EffectiveContract(defaults, nil)
	if err != nil {
		return fail(err)
	}
	data, err := os.ReadFile(*traces)
	if err != nil {
		return fail(err)
	}
	spans, err := core.ParseOTLP(data)
	if err != nil {
		return fail(err)
	}
	for i := range spans { // adapters: no-op on canonical events, converts bench records (R-056)
		for j := range spans[i].Events {
			if e := &spans[i].Events[j]; e.Name == core.FaultEventName {
				e.Attrs = adapters.Convert(e.Attrs)
			}
		}
	}
	runs := core.GroupRuns(n, spans)
	keys := make([]string, 0, len(runs))
	for k := range runs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var recs, bad bytes.Buffer
	for _, k := range keys {
		var rec *core.RunRecord
		var warns []string
		c, inv := contractFor(*catalog, defaults, def, runs[k])
		if inv == nil {
			rec, warns, inv = core.ScoreRun(n, c, runs[k])
		}
		for _, w := range warns {
			fmt.Fprintf(stderr, "warning: run %s: %s\n", k, w)
		}
		if inv != nil {
			fmt.Fprintf(stdout, "run %s: invalid: %s\n", k, inv.Reason)
			b, _ := json.Marshal(map[string]string{"run_id": k, "reason": inv.Reason})
			bad.Write(append(b, '\n'))
			continue
		}
		b, err := json.Marshal(rec)
		if err != nil {
			return fail(err)
		}
		recs.Write(append(b, '\n'))
		fmt.Fprintf(stdout, "run %s: ok\n", k)
	}
	// sidecar first: if a write fails, fail removes it and no half result stays behind
	if err := os.WriteFile(sidecar, bad.Bytes(), 0o644); err != nil {
		return fail(err)
	}
	if err := os.WriteFile(*out, recs.Bytes(), 0o644); err != nil {
		return fail(err)
	}
	return 0
}
