// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package pkg

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/auditbeat/core"
	abtest "github.com/elastic/beats/auditbeat/testing"
	"github.com/elastic/beats/libbeat/logp"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
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
		checkFieldValue(t, event, "system.audit.package.size", uint64(269))
		checkFieldValue(t, event, "system.audit.package.summary", "Test Package")
		checkFieldValue(t, event, "system.audit.package.url", "https://www.elastic.co/")
		checkFieldValue(t, event, "system.audit.package.version", "8.2.0-1ubuntu2~18.04")
	}
}

func TestDpkgInstalledSize(t *testing.T) {
	expected := map[string]uint64{
		"libquadmath0":      269,
		"python-apt-common": 248,
		"libnpth0":          32,
		"bind9-host":        17,
		"libpam-runtime":    300,
		"libsepol1-dev":     1739 * 1024,
		"libisl19":          17 * 1024 * 1024,
		"netbase":           0,
		"python2.7-minimal": 0,
	}

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
	dpkgPath, err = filepath.Abs("testdata/dpkg-size/")
	if err != nil {
		t.Fatal(err)
	}

	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	defer f.(*MetricSet).bucket.DeleteBucket()

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	got := make(map[string]uint64, len(events))
	for _, ev := range events {
		event := mbtest.StandardizeEvent(f, ev, core.AddDatasetToEvent)
		name, err := event.GetValue("system.audit.package.name")
		if err != nil {
			t.Fatal(err)
		}
		size, err := event.GetValue("system.audit.package.size")
		if err != nil {
			size = uint64(0)
		}
		if !assert.IsType(t, "", name) {
			t.Fatal("string expected")
		}
		if !assert.IsType(t, uint64(0), size) {
			t.Fatal("uint64 expected")
		}
		got[name.(string)] = size.(uint64)
	}
	assert.Equal(t, expected, got)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":   "system",
		"datasets": []string{"package"},
	}
}
