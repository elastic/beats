// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && (amd64 || arm64) && cgo

package kerneltracingprovider

import (
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// / Stats tracks the quark internal stats, which are integrated into the beats monitoring runtime
type Stats struct {
	Insertions      *monitoring.Uint
	Removals        *monitoring.Uint
	Aggregations    *monitoring.Uint
	NonAggregations *monitoring.Uint
	Lost            *monitoring.Uint
}

// / NewStats creates a new stats object
func NewStats(reg *monitoring.Registry) *Stats {
	return &Stats{
		Insertions:      monitoring.NewUint(reg, "insertions"),
		Removals:        monitoring.NewUint(reg, "removals"),
		Aggregations:    monitoring.NewUint(reg, "aggregations"),
		NonAggregations: monitoring.NewUint(reg, "nonaggregations"),
		Lost:            monitoring.NewUint(reg, "lost"),
	}
}
