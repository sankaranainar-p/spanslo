package core

import (
	"os"
	"reflect"
	"slices"
	"testing"
)

// R-009: fixtures with parallel calls at identical start times and reasoning only turns.
func TestIndexFixtures(t *testing.T) {
	n := idxNames(t)
	for _, tc := range []struct {
		file         string
		chats, calls int
		want         map[string][2]int
	}{
		{"parallel-identical-start.json", 2, 4, map[string][2]int{
			"0000000000000004": {1, 0}, "0000000000000002": {1, 1},
			"0000000000000003": {1, 2}, "00000000000000a5": {2, 0},
		}},
		{"reasoning-only-turns.json", 4, 2, map[string][2]int{
			"00000000000000b1": {1, 0}, "00000000000000b3": {3, 0},
		}},
	} {
		data, err := os.ReadFile("../testdata/traces/indexing/" + tc.file)
		if err != nil {
			t.Fatal(err)
		}
		spans, err := ParseOTLP(data)
		if err != nil {
			t.Fatalf("%s: %v", tc.file, err)
		}
		for _, order := range []string{"as written", "reversed"} {
			if order == "reversed" {
				slices.Reverse(spans)
			}
			runs := GroupRuns(spans)
			if len(runs) != 1 {
				t.Fatalf("%s: got %d runs, want 1", tc.file, len(runs))
			}
			for _, run := range runs {
				idx := mustIndex(t, n, run)
				if len(idx.Chats) != tc.chats || len(idx.Calls) != tc.calls {
					t.Errorf("%s (%s): got %d chats, %d calls, want %d, %d",
						tc.file, order, len(idx.Chats), len(idx.Calls), tc.chats, tc.calls)
				}
				if got := stepsOf(idx); !reflect.DeepEqual(got, tc.want) {
					t.Errorf("%s (%s): got %v, want %v", tc.file, order, got, tc.want)
				}
			}
		}
	}
}
