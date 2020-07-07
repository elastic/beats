// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package configrequest

import "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"

const (
	// StepRun is a name of Start program event
	StepRun = "sc-run"
	// StepRemove is a name of Remove program event causing beat in version to be uninstalled
	StepRemove = "sc-remove"

	// MetaConfigKey is key used to store configuration in metadata
	MetaConfigKey = "config"
)

// Step is a step needed to be applied
type Step struct {
	// ID identifies kind of operation needed to be executed
	ID string
	// Version is a version of a program
	Version string
	// Spec for the program
	ProgramSpec program.Spec
	// Meta contains additional data such as version, configuration or tags.
	Meta map[string]interface{}
}

func (s *Step) String() string {
	return "[ID:" + s.ID + ", PROCESS: " + s.ProgramSpec.Cmd + " VERSION:" + s.Version + "]"
}
