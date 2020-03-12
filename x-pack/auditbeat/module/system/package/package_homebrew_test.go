// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package pkg

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/auditbeat/core"
	abtest "github.com/elastic/beats/v7/auditbeat/testing"
	"github.com/elastic/beats/v7/libbeat/beat"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestHomebrew(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	oldPath := homebrewCellarPath
	defer func() {
		homebrewCellarPath = oldPath
	}()
	homebrewCellarPath = "testdata/homebrew/"

	// Test just listBrewPackages()
	packages, err := listBrewPackages()
	assert.NoError(t, err)
	if assert.Len(t, packages, 1) {
		pkg := packages[0]
		assert.Equal(t, "test-package", pkg.Name)
		assert.Equal(t, "Test package", pkg.Summary)
		assert.Equal(t, "https://www.elastic.co/", pkg.URL)
		assert.Equal(t, "1.0.0", pkg.Version)
	}

	// Test whole dataset if on Darwin
	if runtime.GOOS == "darwin" {
		f := mbtest.NewReportingMetricSetV2(t, getConfig())
		defer f.(*MetricSet).bucket.DeleteBucket()

		events, errs := mbtest.ReportingFetchV2(f)
		if len(errs) > 0 {
			t.Fatalf("received error: %+v", errs[0])
		}

		if assert.Len(t, events, 1) {
			event := mbtest.StandardizeEvent(f, events[0], core.AddDatasetToEvent)
			checkFieldValue(t, event, "system.audit.package.name", "test-package")
			checkFieldValue(t, event, "system.audit.package.summary", "Test package")
			checkFieldValue(t, event, "system.audit.package.url", "https://www.elastic.co/")
			checkFieldValue(t, event, "system.audit.package.version", "1.0.0")
			checkFieldValue(t, event, "system.audit.package.entity_id", "Krm421rtYM4wgq1S")
		}
	}
}

func checkFieldValue(t *testing.T, event beat.Event, fieldName string, fieldValue interface{}) {
	value, err := event.GetValue(fieldName)
	if assert.NoError(t, err) {
		assert.Equal(t, fieldValue, value)
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

	// Test whole dataset if on Darwin
	if runtime.GOOS == "darwin" {
		f := mbtest.NewReportingMetricSetV2(t, getConfig())
		defer f.(*MetricSet).bucket.DeleteBucket()

		events, errs := mbtest.ReportingFetchV2(f)
		if len(errs) > 0 {
			t.Fatalf("received error: %+v", errs[0])
		}
		assert.Empty(t, events)
	}
}
