// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"time"
)

func EventsMapping(results consumption.ForecastsListResult, report mb.ReporterV2) error {
	event := mb.Event{
		ModuleFields:    common.MapStr{},
		MetricSetFields: common.MapStr{},
		Timestamp:       time.Now(),
	}
	report.Event(event)
	return nil
}
