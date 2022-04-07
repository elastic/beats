// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package pkg

import (
	"bytes"
	"encoding/gob"
	"errors"
	"flag"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/auditbeat/core"
	abtest "github.com/elastic/beats/v8/auditbeat/testing"
	"github.com/elastic/beats/v8/libbeat/logp"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
)

var flagUpdateGob = flag.Bool("update-gob", false, "update persisted gob testdata")

func TestData(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("FIXME: https://github.com/elastic/beats/issues/18855")
	}

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
		checkFieldValue(t, event, "package.name", "test")
		checkFieldValue(t, event, "package.size", uint64(269))
		checkFieldValue(t, event, "package.description", "Test Package")
		checkFieldValue(t, event, "package.reference", "https://www.elastic.co/")
		checkFieldValue(t, event, "package.version", "8.2.0-1ubuntu2~18.04")
		checkFieldValue(t, event, "package.type", "dpkg")
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

func TestPackageGobEncodeDecode(t *testing.T) {
	pkg := Package{
		Name:        "foo",
		Version:     "1.2.3",
		Release:     "1",
		Arch:        "amd64",
		License:     "bar",
		InstallTime: time.Unix(1591021924, 0).UTC(),
		Size:        1234,
		Summary:     "Foo stuff",
		URL:         "http://foo.example.com",
		Type:        "rpm",
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(pkg); err != nil {
		t.Fatal(err)
	}

	const gobTestFile = "testdata/package.v1.gob"
	if *flagUpdateGob {
		// NOTE: If you are updating this file then you may have introduced a
		// a breaking change.
		if err := ioutil.WriteFile(gobTestFile, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("decode", func(t *testing.T) {
		var pkgDecoded Package
		if err := gob.NewDecoder(buf).Decode(&pkgDecoded); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, pkg, pkgDecoded)
	})

	// Validate that we get the same result when decoding an earlier saved version.
	// This detects breakages to the struct or to the encoding/decoding pkgs.
	t.Run("decode_from_file", func(t *testing.T) {
		contents, err := ioutil.ReadFile(gobTestFile)
		if err != nil {
			t.Fatal(err)
		}

		var pkgDecoded Package
		if err := gob.NewDecoder(bytes.NewReader(contents)).Decode(&pkgDecoded); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, pkg, pkgDecoded)
	})
}

// Regression test for https://github.com/elastic/beats/issues/18536 to verify
// that error isn't made public.
func TestPackageWithErrorGobEncode(t *testing.T) {
	pkg := Package{
		error: errors.New("test"),
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(pkg); err != nil {
		t.Fatal(err)
	}
}
