// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

const (
	capabilitiesFilename = "capabilities.yml"
)

// Capability provides a way of applying predefined filter to object.
// It's up to capability to determine if capability is applicable on object.
type Capability interface {
	// Apply applies capabilities on input and returns true if input should be completely blocked
	// otherwise, false and updated input is returned
	Apply(interface{}) (bool, interface{})
}

type capabilitiesManager struct {
	caps []Capability
}

// LoadCapabilities loads capabilities files and prepares manager.
func LoadCapabilities() (Capability, error) {
	handlers := []func(ruler) Capability{
		NewInputCapability,
		NewOutputCapability,
		NewUpgradeCapability,
	}

	var rules ruleDefinitions
	var caps []Capability
	// TODO: load capabilities filter

	// make list of handlers out of capabilities definition
	for _, r := range rules {
		for _, h := range handlers {
			var match bool
			if c := h(r); c != nil {
				caps = append(caps, c)
				match = true
			}
			if !match {
				// TODO: log failure in recognizing rule
			}
		}
	}

	return &capabilitiesManager{
		caps: caps,
	}, nil
}

func (mgr *capabilitiesManager) Apply(in interface{}) (bool, interface{}) {
	var blocked bool
	for _, cap := range mgr.caps {
		blocked, in = cap.Apply(in)
		if blocked {
			return blocked, in
		}
	}

	return false, in
}
