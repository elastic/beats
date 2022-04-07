// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package host

import (
	"testing"

	"github.com/elastic/beats/v8/auditbeat/core"
	abtest "github.com/elastic/beats/v8/auditbeat/testing"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	f := mbtest.NewReportingMetricSetV2(t, getConfig())
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

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"host"},
	}
}
