// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"os"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"gopkg.in/yaml.v2"
)

// Capability provides a way of applying predefined filter to object.
// It's up to capability to determine if capability is applicable on object.
type Capability interface {
	// Apply applies capabilities on input and returns true if input should be completely blocked
	// otherwise, false and updated input is returned
	Apply(interface{}) (bool, interface{})
}

type capabilitiesManager struct {
	caps     []Capability
	reporter status.Reporter
}

// Load loads capabilities files and prepares manager.
func Load(capsFile string, log *logger.Logger, sc status.Controller) (Capability, error) {
	handlers := []func(*logger.Logger, ruleDefinitions, status.Reporter) (Capability, error){
		newInputsCapability,
		newOutputsCapability,
		newUpgradesCapability,
	}

	cm := &capabilitiesManager{
		caps:     make([]Capability, 0),
		reporter: sc.Register("capabilities"),
	}

	// load capabilities from filen()
	fd, err := os.Open(capsFile)
	if err != nil && !os.IsNotExist(err) {
		return cm, err
	}

	if os.IsNotExist(err) {
		log.Infof("capabilities file not found in %s", capsFile)
		return cm, nil
	}
	defer fd.Close()

	var definitions ruleDefinitions
	dec := yaml.NewDecoder(fd)
	if err := dec.Decode(&definitions); err != nil {
		return cm, err
	}

	// make list of handlers out of capabilities definition
	for _, h := range handlers {
		cap, err := h(log, definitions, cm.reporter)
		if err != nil {
			return nil, err
		}

		if cap == nil {
			continue
		}

		cm.caps = append(cm.caps, cap)
	}

	return cm, nil
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
