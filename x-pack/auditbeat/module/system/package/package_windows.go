// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package pkg

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system"
)

const (
	metricsetName = "package"
)

func init() {
	mb.Registry.MustAddMetricSet(system.ModuleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// New returns an error.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return nil, fmt.Errorf("the %v/%v dataset is not supported on Windows", system.ModuleName, metricsetName)
}
