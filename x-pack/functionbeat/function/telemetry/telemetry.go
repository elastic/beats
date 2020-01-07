// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package telemetry

import (
	"strconv"
	"sync"

	"github.com/elastic/beats/libbeat/monitoring"
)

// T is a telemetry instance
type T interface {
	AddTriggeredFunction(typeName string, triggerCount int64, eventCount int64)
}

type telemetry struct {
	sync.Mutex
	nextID int

	registry *monitoring.Registry
}

// New returns a new telemetry registry.
func New(r *monitoring.Registry) T {
	return &telemetry{
		nextID:   0,
		registry: r.NewRegistry("functions"),
	}
}

// Ignored is used when the package is not monitored.
func Ignored() T {
	return nil
}

// AddTriggeredFunction adds a triggered function data to the registry.
func (t *telemetry) AddTriggeredFunction(typeName string, triggerCount, eventCount int64) {
	r := t.createFunctionRegistry()

	monitoring.NewString(r, "type").Set(typeName)
	monitoring.NewInt(r, "trigger_count").Set(triggerCount)
	monitoring.NewInt(r, "event_count").Set(eventCount)
}

func (t *telemetry) createFunctionRegistry() *monitoring.Registry {
	t.Lock()
	defer t.Unlock()

	r := t.registry.NewRegistry(strconv.Itoa(t.nextID))
	t.nextID++

	return r
}
