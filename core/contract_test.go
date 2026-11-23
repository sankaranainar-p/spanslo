package core

import (
	"os"
	"reflect"
	"testing"
)

const testDefaults = `
contract:
  safety:
    must_not_call_tags: [destructive, outbound]
  efficiency:
    max_retries_same_call: 3
    max_retries_same_tool: 5
`

// R-025: a key set in the fault file overrides the default; omitted keys keep it.
func TestEffectiveContract(t *testing.T) {
	for name, tc := range map[string]struct {
		fault string
		want  Contract
	}{
		"no contract": {"id: x\n", Contract{[]string{"destructive", "outbound"}, 3, 5}},
		"empty file":  {"", Contract{[]string{"destructive", "outbound"}, 3, 5}},
		"one key": {"contract:\n  efficiency:\n    max_retries_same_tool: 9\n",
			Contract{[]string{"destructive", "outbound"}, 3, 9}},
		"zero and empty list": {"contract:\n  safety:\n    must_not_call_tags: []\n  efficiency:\n    max_retries_same_call: 0\n",
			Contract{[]string{}, 0, 5}},
		"all keys": {"contract:\n  safety:\n    must_not_call_tags: [outbound]\n  efficiency:\n    max_retries_same_call: 1\n    max_retries_same_tool: 2\n",
			Contract{[]string{"outbound"}, 1, 2}},
	} {
		got, err := EffectiveContract([]byte(testDefaults), []byte(tc.fault))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !reflect.DeepEqual(*got, tc.want) {
			t.Errorf("%s: got %+v, want %+v", name, *got, tc.want)
		}
	}
}

func TestEffectiveContractErrors(t *testing.T) {
	if _, err := EffectiveContract([]byte("contract: {}\n"), nil); err == nil {
		t.Error("defaults without keys: want error")
	}
	if _, err := EffectiveContract([]byte(testDefaults), []byte("contract: [")); err == nil {
		t.Error("bad fault yaml: want error")
	}
}

// R-025: the real catalog files give the defaults, since no fault overrides them yet.
func TestEffectiveContractCatalog(t *testing.T) {
	defaults, err := os.ReadFile("../catalog/defaults.yaml")
	if err != nil {
		t.Fatal(err)
	}
	fault, err := os.ReadFile("../catalog/tool-timeout-30s.yaml")
	if err != nil {
		t.Fatal(err)
	}
	got, err := EffectiveContract(defaults, fault)
	if err != nil {
		t.Fatal(err)
	}
	want := Contract{[]string{"destructive", "outbound"}, 3, 5}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("got %+v, want %+v", *got, want)
	}
}
