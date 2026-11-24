package core

import "slices"

// Percentiles is p50 and p95 of one SLI, reported with the count of unrecovered episodes
// (R-046). OK is false when no value is left: the percentiles are undefined (R-066).
type Percentiles struct {
	P50, P95    float64
	OK          bool
	Unrecovered int
}

// percentile is the linear interpolation percentile p (0 to 1) of sorted values; at 0.5 it
// equals Median.
// ponytail: R-046 names no percentile method; linear interpolation, swap here if design.md says nearest rank.
func percentile(sorted []float64, p float64) float64 {
	h := p * float64(len(sorted)-1)
	lo := int(h)
	if lo+1 >= len(sorted) {
		return sorted[lo]
	}
	return sorted[lo] + (h-float64(lo))*(sorted[lo+1]-sorted[lo])
}

func summarize(vals []float64, unrecovered int) Percentiles {
	p := Percentiles{Unrecovered: unrecovered}
	if len(vals) == 0 {
		return p
	}
	v := slices.Clone(vals)
	slices.Sort(v)
	p.P50, p.P95, p.OK = percentile(v, 0.5), percentile(v, 0.95), true
	return p
}

// SLIPercentiles holds the percentiles of the four episode SLIs and of taf and
// turn_amplification. Every field carries the same unrecovered episode count.
type SLIPercentiles struct {
	TurnsToRecover, TimeToRecoverMS, CascadeSpans, RecoveryTokens Percentiles
	TAF, TurnAmplification                                        Percentiles
}

// ComputePercentiles reports p50 and p95 over the triggered runs (R-035, R-046). Unrecovered
// episodes are left out of the four episode SLIs and counted in Unrecovered. taf and
// turn_amplification are per run and skip runs where they are undefined (R-066).
func ComputePercentiles(runs []*RunRecord, baselines map[BaselineKey][]*RunRecord) SLIPercentiles {
	var turns, ttr, cascade, tokens, taf, amp []float64
	unrecovered := 0
	for _, r := range runs {
		if !IsTriggered(r) {
			continue
		}
		for _, e := range r.Episodes {
			if !e.SurfaceRecovered {
				unrecovered++
				continue
			}
			cascade = append(cascade, float64(e.CascadeSpans))
			tokens = append(tokens, float64(e.RecoveryTokens))
			if e.TurnsToRecover != nil {
				turns = append(turns, float64(*e.TurnsToRecover))
			}
			if e.TimeToRecoverMS != nil {
				ttr = append(ttr, float64(*e.TimeToRecoverMS))
			}
		}
		if v := TAF(r, baselines); v.OK {
			taf = append(taf, v.Value)
		}
		if v := TurnAmplification(r, baselines); v.OK {
			amp = append(amp, v.Value)
		}
	}
	return SLIPercentiles{
		summarize(turns, unrecovered), summarize(ttr, unrecovered),
		summarize(cascade, unrecovered), summarize(tokens, unrecovered),
		summarize(taf, unrecovered), summarize(amp, unrecovered),
	}
}
