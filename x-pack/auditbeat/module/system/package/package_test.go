// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package pkg

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/auditbeat/core"
	abtest "github.com/elastic/beats/v7/auditbeat/testing"
	"github.com/elastic/beats/v7/libbeat/logp"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	defer f.(*MetricSet).bucket.DeleteBucket()

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	if len(events) == 0 {
		t.Fatal("no events were generated")
	}

	fullEvent := mbtest.StandardizeEvent(f, events[len(events)-1], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, fullEvent, "")
}

func TestDpkg(t *testing.T) {
	logp.TestingSetup()

	defer abtest.SetupDataDir(t)()

	// Disable all except dpkg
	rpmPathOld := rpmPath
	dpkgPathOld := dpkgPath
	brewPathOld := homebrewCellarPath
	defer func() {
		rpmPath = rpmPathOld
		dpkgPath = dpkgPathOld
		homebrewCellarPath = brewPathOld
	}()
	rpmPath = "/does/not/exist"
	homebrewCellarPath = "/does/not/exist"

	var err error
	dpkgPath, err = filepath.Abs("testdata/dpkg/")
	if err != nil {
		t.Fatal(err)
	}

	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	defer f.(*MetricSet).bucket.DeleteBucket()

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	if assert.Len(t, events, 1) {
		event := mbtest.StandardizeEvent(f, events[0], core.AddDatasetToEvent)
		checkFieldValue(t, event, "system.audit.package.name", "test")
		checkFieldValue(t, event, "system.audit.package.summary", "Test Package")
		checkFieldValue(t, event, "system.audit.package.url", "https://www.elastic.co/")
		checkFieldValue(t, event, "system.audit.package.version", "8.2.0-1ubuntu2~18.04")
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":   "system",
		"datasets": []string{"package"},
	}
}
