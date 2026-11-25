package core

import (
	"testing"
	"time"
)

const sloOK = `slos:
  - {name: recovery, sli: surface_recovery_rate, target: 0.9}
  - {name: unsafe, sli: unsafe_action_rate, target_max: 0.01}
  - {name: ttr, sli: time_to_recover, threshold_max: 30000, objective: 0.95}
  - {name: rr, sli: resilience_ratio, target: 0.8, burn: false}
windows:
  time:
    budget: 30d
    burn:
      - {long: 1h, short: 5m, factor: 14.4}
  count:
    budget: 1000
    burn:
      - {long: 100, short: 10, factor: 6}
`

// R-048: a valid file parses; burn defaults to true; all three forms and both window kinds are read.
func TestParseSLOFile(t *testing.T) {
	f, err := ParseSLOFile([]byte(sloOK))
	if err != nil {
		t.Fatal(err)
	}
	s := f.SLOs
	if len(s) != 4 || s[0].Target == nil || *s[0].Target != 0.9 || s[1].TargetMax == nil || *s[1].TargetMax != 0.01 ||
		s[2].ThresholdMax == nil || *s[2].ThresholdMax != 30000 || s[2].Objective == nil || *s[2].Objective != 0.95 {
		t.Fatalf("got %+v", s)
	}
	if !s[0].BurnEnabled() || !s[1].BurnEnabled() || !s[2].BurnEnabled() || s[3].BurnEnabled() {
		t.Error("burn default or explicit false wrong")
	}
	tw, cw := f.Windows.Time, f.Windows.Count
	if tw == nil || tw.Budget != 720*time.Hour || len(tw.Burn) != 1 ||
		tw.Burn[0] != (TimeBurn{time.Hour, 5 * time.Minute, 14.4}) {
		t.Errorf("time windows: %+v", tw)
	}
	if cw == nil || cw.Budget != 1000 || len(cw.Burn) != 1 || cw.Burn[0] != (CountBurn{100, 10, 6}) {
		t.Errorf("count windows: %+v", cw)
	}
}

// R-048: only the time windows, with no burn pairs, is valid.
func TestParseSLOFileTimeOnly(t *testing.T) {
	f, err := ParseSLOFile([]byte("slos:\n  - {name: a, sli: x, target: 1.0}\nwindows:\n  time: {budget: 1h}\n"))
	if err != nil || f.Windows.Count != nil || f.Windows.Time == nil || f.Windows.Time.Budget != time.Hour {
		t.Errorf("got %+v, %v", f, err)
	}
}

// R-048: two forms, no form, a half form, a bad value, unknown keys and bad windows are rejected.
func TestParseSLOFileRejects(t *testing.T) {
	const one = "slos:\n  - {name: a, sli: x, target: 0.9}\n"
	const win = "windows:\n  time: {budget: 1h}\n"
	mk := func(s string) string { return "slos:\n  - {name: a, sli: x, " + s + "}\n" + win }
	cases := map[string]string{
		"two forms":                   mk("target: 0.9, target_max: 0.1"),
		"no form":                     mk("burn: false"),
		"threshold without objective": mk("threshold_max: 5"),
		"objective without threshold": mk("target: 0.9, objective: 0.9"),
		"objective above 1":           mk("threshold_max: 5, objective: 1.5"),
		"negative target":             mk("target: -0.1"),
		"unknown key":                 mk("target: 0.9, extra: 1"),
		"burn not a boolean":          mk("target: 0.9, burn: maybe"),
		"missing name":                "slos:\n  - {sli: x, target: 0.9}\n" + win,
		"empty file":                  "",
		"not yaml":                    "slos: [",
		"no windows":                  one,
		"empty windows":               one + "windows: {}\n",
		"bad duration":                one + "windows:\n  time: {budget: soon}\n",
		"zero budget":                 one + "windows:\n  time: {budget: 0s}\n",
		"negative budget":             one + "windows:\n  time: {budget: -1h}\n",
		"short above long":            one + "windows:\n  time: {budget: 1h, burn: [{long: 5m, short: 1h, factor: 2}]}\n",
		"no factor":                   one + "windows:\n  time: {budget: 1h, burn: [{long: 1h, short: 5m}]}\n",
		"count budget zero":           one + "windows:\n  count: {budget: 0}\n",
		"count budget fraction":       one + "windows:\n  count: {budget: 1.5}\n",
		"count short above long":      one + "windows:\n  count: {budget: 100, burn: [{long: 5, short: 10, factor: 2}]}\n",
		"count duration unit":         one + "windows:\n  count: {budget: 100, burn: [{long: 1h, short: 5, factor: 2}]}\n",
		"no slos":                     win,
	}
	for name, in := range cases {
		if f, err := ParseSLOFile([]byte(in)); err == nil {
			t.Errorf("%s: got %+v, want an error", name, f)
		}
	}
}
