// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"github.com/elastic/beats/metricbeat/mb"
)

// AzureService interface for the azure monitor service and mock for testing
type ClientInterface interface {
	InitResources(report mb.ReporterV2) error
	GetMetricValues(report mb.ReporterV2) error
	GetResources() ResourceConfiguration
}
