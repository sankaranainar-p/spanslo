package core

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// SLO is one entry of the slos list (R-048). Exactly one form is set: Target, TargetMax, or
// ThresholdMax with Objective. Burn is nil when absent, which means true.
type SLO struct {
	Name         string   `yaml:"name"`
	SLI          string   `yaml:"sli"`
	Target       *float64 `yaml:"target"`
	TargetMax    *float64 `yaml:"target_max"`
	ThresholdMax *float64 `yaml:"threshold_max"`
	Objective    *float64 `yaml:"objective"`
	Burn         *bool    `yaml:"burn"`
}

// BurnEnabled is the burn setting with its default of true (R-048).
func (s SLO) BurnEnabled() bool { return s.Burn == nil || *s.Burn }

// TimeBurn is one pair of long and short time windows; an alert fires when both burn rates
// reach Factor (R-053).
type TimeBurn struct {
	Long, Short time.Duration
	Factor      float64
}

// CountBurn is the same pair for count windows, in runs.
type CountBurn struct {
	Long, Short int
	Factor      float64
}

// TimeWindows is windows.time.
type TimeWindows struct {
	Budget time.Duration
	Burn   []TimeBurn
}

// CountWindows is windows.count; Budget is a number of runs.
type CountWindows struct {
	Budget int
	Burn   []CountBurn
}

// Windows holds the configured window kinds; a nil kind is not configured.
type Windows struct {
	Time  *TimeWindows
	Count *CountWindows
}

// SLOFile is a parsed SLO file (R-048).
type SLOFile struct {
	SLOs    []SLO
	Windows Windows
}

type rawFile struct {
	SLOs    []SLO `yaml:"slos"`
	Windows struct {
		Time *struct {
			Budget string `yaml:"budget"`
			Burn   []struct {
				Long   string  `yaml:"long"`
				Short  string  `yaml:"short"`
				Factor float64 `yaml:"factor"`
			} `yaml:"burn"`
		} `yaml:"time"`
		// run counts are read as floats so 1.5 is rejected, not truncated by the decoder
		Count *struct {
			Budget float64 `yaml:"budget"`
			Burn   []struct {
				Long   float64 `yaml:"long"`
				Short  float64 `yaml:"short"`
				Factor float64 `yaml:"factor"`
			} `yaml:"burn"`
		} `yaml:"count"`
	} `yaml:"windows"`
}

// ParseSLOFile reads an SLO file and rejects what R-048 rules out: an SLO without exactly one
// form, unknown keys, and a bad window. Callers pass the file bytes (R-008).
// ponytail: ranges are checked from R-048 and R-067, not from spec/slo.schema.json; durations take
// Go units plus a d suffix for days. Compare with the schema if a valid file is rejected.
func ParseSLOFile(data []byte) (*SLOFile, error) {
	var raw rawFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err == io.EOF {
		return nil, fmt.Errorf("slo file: empty")
	} else if err != nil {
		return nil, fmt.Errorf("slo file: %w", err)
	}
	if len(raw.SLOs) == 0 {
		return nil, fmt.Errorf("slo file: slos is missing or empty")
	}
	for i, s := range raw.SLOs {
		if err := validateSLO(s); err != nil {
			return nil, fmt.Errorf("slos[%d]: %w", i, err)
		}
	}
	w := raw.Windows
	if w.Time == nil && w.Count == nil {
		return nil, fmt.Errorf("slo file: windows needs time or count")
	}
	f := &SLOFile{SLOs: raw.SLOs}
	if t := w.Time; t != nil {
		tw := &TimeWindows{}
		var err error
		if tw.Budget, err = parseDur("windows.time.budget", t.Budget); err != nil {
			return nil, err
		}
		for i, b := range t.Burn {
			p := fmt.Sprintf("windows.time.burn[%d]", i)
			tb := TimeBurn{Factor: b.Factor}
			if tb.Long, err = parseDur(p+".long", b.Long); err != nil {
				return nil, err
			}
			if tb.Short, err = parseDur(p+".short", b.Short); err != nil {
				return nil, err
			}
			if tb.Short > tb.Long || !(tb.Factor > 0) {
				return nil, fmt.Errorf("%s: want short at most long and factor above 0", p)
			}
			tw.Burn = append(tw.Burn, tb)
		}
		f.Windows.Time = tw
	}
	if c := w.Count; c != nil {
		cw := &CountWindows{}
		var err error
		if cw.Budget, err = wholeRuns("windows.count.budget", c.Budget); err != nil {
			return nil, err
		}
		if cw.Budget < 1 {
			return nil, fmt.Errorf("windows.count.budget: want at least 1 run, got %d", cw.Budget)
		}
		for i, b := range c.Burn {
			p := fmt.Sprintf("windows.count.burn[%d]", i)
			cb := CountBurn{Factor: b.Factor}
			if cb.Long, err = wholeRuns(p+".long", b.Long); err != nil {
				return nil, err
			}
			if cb.Short, err = wholeRuns(p+".short", b.Short); err != nil {
				return nil, err
			}
			if cb.Short < 1 || cb.Short > cb.Long || !(cb.Factor > 0) {
				return nil, fmt.Errorf("%s: want 1 <= short <= long and factor above 0", p)
			}
			cw.Burn = append(cw.Burn, cb)
		}
		f.Windows.Count = cw
	}
	return f, nil
}

// wholeRuns converts a run count read as a float; a fraction is an error, not truncated.
func wholeRuns(field string, v float64) (int, error) {
	if v != math.Trunc(v) || v < 0 || v > math.MaxInt32 {
		return 0, fmt.Errorf("%s: %v is not a whole number of runs", field, v)
	}
	return int(v), nil
}

func parseDur(field, s string) (time.Duration, error) {
	var d time.Duration
	var err error
	if n, ok := strings.CutSuffix(s, "d"); ok {
		var days float64
		days, err = strconv.ParseFloat(n, 64)
		d = time.Duration(days * float64(24*time.Hour))
	} else {
		d, err = time.ParseDuration(s)
	}
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s: %q is not a positive duration", field, s)
	}
	return d, nil
}

func validateSLO(s SLO) error {
	if s.Name == "" || s.SLI == "" {
		return fmt.Errorf("name and sli are required")
	}
	forms := []*float64{s.Target, s.TargetMax, s.ThresholdMax}
	n := 0
	for _, v := range forms {
		if v != nil {
			n++
			if !(*v >= 0) {
				return fmt.Errorf("%s: %v must be at least 0", s.Name, *v)
			}
		}
	}
	if n != 1 {
		return fmt.Errorf("%s: want exactly one of target, target_max, threshold_max, got %d", s.Name, n)
	}
	if (s.ThresholdMax != nil) != (s.Objective != nil) {
		return fmt.Errorf("%s: objective goes with threshold_max and with no other form", s.Name)
	}
	if o := s.Objective; o != nil && !(*o >= 0 && *o <= 1) {
		return fmt.Errorf("%s: objective %v must be between 0 and 1", s.Name, *o)
	}
	return nil
}
