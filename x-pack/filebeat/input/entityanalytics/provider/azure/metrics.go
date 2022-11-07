// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import "github.com/elastic/elastic-agent-libs/monitoring"

type inputMetrics struct {
	id       string
	registry *monitoring.Registry

	fullSyncTotal            *monitoring.Uint
	fullSyncSuccess          *monitoring.Uint
	fullSyncFailure          *monitoring.Uint
	incrementalUpdateTotal   *monitoring.Uint
	incrementalUpdateSuccess *monitoring.Uint
	incrementalUpdateFailure *monitoring.Uint
}

func (m *inputMetrics) Close() {
	m.registry.Remove(m.id)
}

func newMetrics(registry *monitoring.Registry, id string) *inputMetrics {
	reg := registry.NewRegistry(id)

	monitoring.NewString(reg, "input").Set(Name)
	monitoring.NewString(reg, "id").Set(id)

	m := inputMetrics{
		id:                       id,
		registry:                 registry,
		fullSyncTotal:            monitoring.NewUint(reg, "sync.full.total"),
		fullSyncSuccess:          monitoring.NewUint(reg, "sync.full.success"),
		fullSyncFailure:          monitoring.NewUint(reg, "sync.full.failure"),
		incrementalUpdateTotal:   monitoring.NewUint(reg, "sync.incremental.total"),
		incrementalUpdateSuccess: monitoring.NewUint(reg, "sync.incremental.success"),
		incrementalUpdateFailure: monitoring.NewUint(reg, "sync.incremental.failure"),
	}

	return &m
}
