// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build cgo

package oracle

import (
	"fmt"

	// Driver
	_ "github.com/godror/godror"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for the "oracle" module.
	if err := mb.Registry.AddModule("oracle", newModule); err != nil {
		panic(err)
	}
}

// newModule adds validation that hosts is non-empty, a requirement to use the
// Oracle module.
func newModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := ConnectionDetails{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, fmt.Errorf("error parsing config module: %w", err)
	}

	return &base, nil
}
