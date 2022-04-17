// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package telemetry

import (
	"github.com/menderesk/beats/v7/libbeat/monitoring"
)

// T is a telemetry instance
type T interface {
	AddTriggeredFunction()
}

type telemetry struct {
	registry       *monitoring.Registry
	countFunctions *monitoring.Int
}

// New returns a new telemetry registry.
func New(r *monitoring.Registry) T {
	return &telemetry{
		registry:       r.NewRegistry("functions"),
		countFunctions: monitoring.NewInt(r, "count"),
	}
}

// Ignored is used when the package is not monitored.
func Ignored() T {
	return nil
}

// AddTriggeredFunction adds a triggered function data to the registry.
func (t *telemetry) AddTriggeredFunction() {
	t.countFunctions.Inc()
}
