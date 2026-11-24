package adapters

import (
	"maps"
	"reflect"
	"testing"
)

// R-057: a bench record converts with every canonical attribute.
func TestConvertBench(t *testing.T) {
	in := Record{"source": "bench", "id": "tool-timeout-30s", "surface": "tool", "target": "search", "turn": int64(2), "call_index": int64(0), "mode": "single", "triggered": true, "virtual_duration_ms": int64(30000)}
	got := Convert(in)
	want := Record{
		"spanslo.fault.id": "tool-timeout-30s", "spanslo.fault.surface": "tool", "spanslo.fault.target": "search",
		"spanslo.fault.turn": int64(2), "spanslo.fault.call_index": int64(0), "spanslo.fault.mode": "single",
		"spanslo.fault.triggered": true, "spanslo.fault.virtual_duration_ms": int64(30000), "spanslo.fault.source": "bench",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// R-058: unknown sources pass through untouched and none is dropped.
func TestConvertUnknownPassthrough(t *testing.T) {
	in := Record{"source": "mystery", "id": "x", "turn": 3}
	orig := maps.Clone(in)
	in2 := Record{"id": "no source"}
	out := ConvertAll([]Record{in, in2})
	if len(out) != 2 || !reflect.DeepEqual(out[0], orig) || !reflect.DeepEqual(out[1], Record{"id": "no source"}) {
		t.Errorf("got %v", out)
	}
}
