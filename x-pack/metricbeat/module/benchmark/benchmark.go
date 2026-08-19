// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package benchmark

import (
	"github.com/elastic/beats/v9/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for the "system" module.
	if err := mb.Registry.AddModule("benchmark", newModule); err != nil {
		panic(err)
	}
}

func newModule(base mb.BaseModule) (mb.Module, error) {
	return &base, nil
}
