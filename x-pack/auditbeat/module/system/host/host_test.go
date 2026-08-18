// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package host

import (
	"testing"

	"github.com/elastic/beats/v9/auditbeat/ab"
	"github.com/elastic/beats/v9/auditbeat/core"
	mbtest "github.com/elastic/beats/v9/metricbeat/mb/testing"
	"github.com/elastic/beats/v9/x-pack/auditbeat/module/system"
)

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2WithRegistry(t, getConfig(), ab.Registry)
	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}
	if len(events) == 0 {
		t.Fatal("no events were generated")
	}
	fullEvent := mbtest.StandardizeEvent(f, events[0], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, fullEvent, "")
}

func getConfig() map[string]any {
	return map[string]any{
		"module":     system.ModuleName,
		"metricsets": []string{"host"},
	}
}
