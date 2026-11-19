package core

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// Contract is the effective contract of one fault: the safety and efficiency keys the
// scorer reads (R-025, R-021, R-023).
type Contract struct {
	MustNotCallTags    []string
	MaxRetriesSameCall int
	MaxRetriesSameTool int
}

// contractDoc reads a catalog file; pointers tell a key that is absent from one set to zero.
type contractDoc struct {
	Contract struct {
		Safety struct {
			MustNotCallTags *[]string `yaml:"must_not_call_tags"`
		} `yaml:"safety"`
		Efficiency struct {
			MaxRetriesSameCall *int `yaml:"max_retries_same_call"`
			MaxRetriesSameTool *int `yaml:"max_retries_same_tool"`
		} `yaml:"efficiency"`
	} `yaml:"contract"`
}

// EffectiveContract lays every key present under the fault file's contract over the
// defaults; omitted keys keep their default (R-025). Callers pass file bytes (R-008).
func EffectiveContract(defaults, fault []byte) (*Contract, error) {
	var d, f contractDoc
	if err := yaml.Unmarshal(defaults, &d); err != nil {
		return nil, fmt.Errorf("defaults: %w", err)
	}
	if err := yaml.Unmarshal(fault, &f); err != nil {
		return nil, fmt.Errorf("fault: %w", err)
	}
	ds, de := d.Contract.Safety, d.Contract.Efficiency
	if ds.MustNotCallTags == nil || de.MaxRetriesSameCall == nil || de.MaxRetriesSameTool == nil {
		return nil, fmt.Errorf("defaults: contract lacks a required key")
	}
	c := &Contract{*ds.MustNotCallTags, *de.MaxRetriesSameCall, *de.MaxRetriesSameTool}
	if v := f.Contract.Safety.MustNotCallTags; v != nil {
		c.MustNotCallTags = *v
	}
	if v := f.Contract.Efficiency.MaxRetriesSameCall; v != nil {
		c.MaxRetriesSameCall = *v
	}
	if v := f.Contract.Efficiency.MaxRetriesSameTool; v != nil {
		c.MaxRetriesSameTool = *v
	}
	return c, nil
}
