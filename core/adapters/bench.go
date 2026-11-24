package adapters

// Record is one fault record as a source emits it, or one canonical fault event after conversion.
type Record map[string]any

// benchKeys are the fields of a bench fault record; each becomes spanslo.fault.<key> (R-057).
var benchKeys = []string{"id", "surface", "target", "turn", "call_index", "mode", "triggered", "virtual_duration_ms"}

// Convert turns a bench record into a canonical fault event with source bench (R-056, R-057).
// A record from any other source is returned as is, same map, neither changed nor dropped (R-058).
// ponytail: bench record layout assumed to use the short field names above; adjust benchKeys if the harness differs.
func Convert(r Record) Record {
	if r["source"] != "bench" {
		return r
	}
	out := Record{"spanslo.fault.source": "bench"}
	for _, k := range benchKeys {
		if v, ok := r[k]; ok {
			out["spanslo.fault."+k] = v
		}
	}
	return out
}

// ConvertAll converts every record in order; the result has the same length (R-058).
func ConvertAll(rs []Record) []Record {
	out := make([]Record, len(rs))
	for i, r := range rs {
		out[i] = Convert(r)
	}
	return out
}
