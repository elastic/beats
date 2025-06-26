// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package events

import (
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Create a new Metricbeat Event object with a random Transaction ID (so it has no predictable relationship to other events outside of @timestamp).
func CreateEventWithRandomTransactionId(info *utils.ClusterInfo, metricSetFields mapstr.M) mb.Event {
	return CreateEvent(info, metricSetFields, utils.NewUUIDV4())
}

// Create a new Metricbeat Event object containing expected fields and the dynamic portion.
func CreateEvent(info *utils.ClusterInfo, metricSetFields mapstr.M, transactionId string) mb.Event {
	return mb.Event{
		MetricSetFields: metricSetFields,
		ModuleFields: mapstr.M{
			"cluster": mapstr.M{
				"id":      info.ClusterID,
				"name":    info.ClusterName,
				"version": info.Version.Number.String(),
			},
			"transactionId": transactionId,
		},
		RootFields: mapstr.M{
			"service.name":             "autoops_es",
			"metricbeatVersion":        version.GetDefaultVersion(),
			"commit":                   version.Commit(),
			"orchestrator.resource.id": utils.GetResourceID(),
		},
	}
}

// Report an event and mark the fraction and total fractions consistently
func ReportEvent(r mb.ReporterV2, event mb.Event, index int, total int) {
	event.ModuleFields["fractionId"] = index
	event.ModuleFields["totalAmountOfFractions"] = total

	r.Event(event)
}

// Report Metricbeat Events marked with the same transaction
func ReportEvents(r mb.ReporterV2, events []mb.Event) {
	var total = len(events)

	for index, event := range events {
		ReportEvent(r, event, index, total)
	}
}

// Create a new Metricbeat Events with a shared, random Transaction ID
func CreateAndReportEventsWithRandomTransactionId(r mb.ReporterV2, info *utils.ClusterInfo, metricSets []mapstr.M) {
	CreateAndReportEvents(r, info, metricSets, utils.NewUUIDV4())
}

// Create a new Metricbeat Events
func CreateAndReportEvents(r mb.ReporterV2, info *utils.ClusterInfo, metricSets []mapstr.M, transactionId string) {
	var total = len(metricSets)

	for index, metricSetFields := range metricSets {
		event := CreateEvent(info, metricSetFields, transactionId)

		ReportEvent(r, event, index, total)
	}
}
