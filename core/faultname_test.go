package core

import (
	"os"
	"testing"

	"go.yaml.in/yaml/v3"
)

// FaultEventName is a constant in core; the mapping is the contract.
// This guard fails if the two drift apart.
func TestFaultEventNameMatchesMapping(t *testing.T) {
	data, err := os.ReadFile("../spec/semconv-mapping.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Spanslo struct {
			FaultEvent struct {
				Name string `yaml:"name"`
			} `yaml:"fault_event"`
		} `yaml:"spanslo_attributes"`
	}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m.Spanslo.FaultEvent.Name != FaultEventName {
		t.Fatalf("mapping fault_event.name = %q, core.FaultEventName = %q", m.Spanslo.FaultEvent.Name, FaultEventName)
	}
}
