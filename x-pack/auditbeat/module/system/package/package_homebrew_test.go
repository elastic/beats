// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package pkg

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/auditbeat/core"
	abtest "github.com/elastic/beats/auditbeat/testing"
	"github.com/elastic/beats/libbeat/beat"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestHomebrew(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	oldPath := homebrewCellarPath
	defer func() {
		homebrewCellarPath = oldPath
	}()
	homebrewCellarPath = "../../../tests/files/homebrew/"

	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	defer f.(*MetricSet).bucket.DeleteBucket()

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}
	assert.Len(t, events, 1)

	event := mbtest.StandardizeEvent(f, events[0], core.AddDatasetToEvent)
	checkFieldValue(t, event, "system.audit.package.name", "test-package")
	checkFieldValue(t, event, "system.audit.package.summary", "Test package")
	checkFieldValue(t, event, "system.audit.package.url", "https://www.elastic.co/")
	checkFieldValue(t, event, "system.audit.package.version", "1.0.0")
	checkFieldValue(t, event, "system.audit.package.installtime", time.Date(2019, 4, 17, 13, 14, 57, 205133721, time.FixedZone("BST", 60*60)))
	checkFieldValue(t, event, "system.audit.package.entity_id", "Krm421rtYM4wgq1S")
}

func checkFieldValue(t *testing.T, event beat.Event, fieldName string, fieldValue interface{}) {
	value, err := event.GetValue(fieldName)
	if assert.NoError(t, err) {
		switch v := value.(type) {
		case time.Time:
			assert.True(t, v.Equal(fieldValue.(time.Time)), "Time is not equal: %+v", v)
		default:
			assert.Equal(t, fieldValue, value)
		}
	}
}

func TestHomebrewNotExist(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	oldPath := homebrewCellarPath
	defer func() {
		homebrewCellarPath = oldPath
	}()
	homebrewCellarPath = "/does/not/exist"

	// Test just listBrewPackages()
	packages, err := listBrewPackages()
	if assert.Error(t, err) {
		assert.True(t, os.IsNotExist(err), "Unexpected error %v", err)
	}
	assert.Empty(t, packages)

	// Test whole dataset
	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	defer f.(*MetricSet).bucket.DeleteBucket()

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}
	assert.Empty(t, events)
}
