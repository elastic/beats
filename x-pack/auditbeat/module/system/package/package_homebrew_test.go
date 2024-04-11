// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package pkg

import (
	"github.com/elastic/beats/v7/auditbeat/ab"
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
	homebrewCellarPath = []string{"testdata/homebrew/"}

	// Test just listBrewPackages()
	packages, err := listBrewPackages("testdata/homebrew/")
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
		f := mbtest.NewReportingMetricSetV2WithRegistry(t, getConfig(), ab.Registry)
		defer deleteBucket(t, f)

		events, errs := mbtest.ReportingFetchV2(f)
		if len(errs) > 0 {
			t.Fatalf("received error: %+v", errs[0])
		}

		if assert.Len(t, events, 1) {
			event := mbtest.StandardizeEvent(f, events[0], core.AddDatasetToEvent)
			checkFieldValue(t, event, "event.kind", "state")
			checkFieldValue(t, event, "event.category", []string{"package"})
			checkFieldValue(t, event, "event.type", []string{"info"})
			checkFieldValue(t, event, "system.audit.package.name", "test-package")
			checkFieldValue(t, event, "system.audit.package.summary", "Test package")
			checkFieldValue(t, event, "system.audit.package.url", "https://www.elastic.co/")
			checkFieldValue(t, event, "system.audit.package.version", "1.0.0")
			// FIXME: The value of this field changes on each execution in CI - https://github.com/elastic/beats/issues/18855
			// checkFieldValue(t, event, "system.audit.package.entity_id", "Krm421rtYM4wgq1S")
			checkFieldValue(t, event, "package.name", "test-package")
			checkFieldValue(t, event, "package.description", "Test package")
			checkFieldValue(t, event, "package.reference", "https://www.elastic.co/")
			checkFieldValue(t, event, "package.version", "1.0.0")
			checkFieldValue(t, event, "package.type", "brew")
		}
	}
}

func checkFieldValue(t *testing.T, event beat.Event, fieldName string, fieldValue interface{}) {
	t.Helper()
	value, err := event.GetValue(fieldName)
	if assert.NoError(t, err, "checking field %s", fieldName) {
		assert.Equal(t, fieldValue, value, "checking field %v", fieldName)
	}
}

func TestHomebrewNotExist(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	oldPath := homebrewCellarPath
	defer func() {
		homebrewCellarPath = oldPath
	}()
	homebrewCellarPath = []string{"/does/not/exist"}

	// Test just listBrewPackages()
	packages, err := listBrewPackages("/does/not/exist")
	if assert.Error(t, err) {
		assert.True(t, os.IsNotExist(err), "Unexpected error %v", err)
	}
	assert.Empty(t, packages)

	// Test whole dataset if on Darwin
	if runtime.GOOS == "darwin" {
		f := mbtest.NewReportingMetricSetV2WithRegistry(t, getConfig(), ab.Registry)
		defer deleteBucket(t, f)

		events, errs := mbtest.ReportingFetchV2(f)
		if len(errs) > 0 {
			t.Fatalf("received error: %+v", errs[0])
		}
		assert.Empty(t, events)
	}
}
