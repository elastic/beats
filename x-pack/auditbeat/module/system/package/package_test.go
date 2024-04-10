// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package pkg

import (
	"bytes"
	"encoding/gob"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/auditbeat/core"
	"github.com/elastic/beats/v7/auditbeat/datastore"
	abtest "github.com/elastic/beats/v7/auditbeat/testing"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestMain(t *testing.M) {
	InitializeModule()
}

func TestData(t *testing.T) {
	defer abtest.SetupDataDir(t)()

	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	defer deleteBucket(t, f)

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
	homebrewCellarPath = []string{"/does/not/exist"}

	var err error
	dpkgPath, err = filepath.Abs("testdata/dpkg/")
	if err != nil {
		t.Fatal(err)
	}

	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	defer deleteBucket(t, f)

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
	homebrewCellarPath = []string{"/does/not/exist"}

	var err error
	dpkgPath, err = filepath.Abs("testdata/dpkg-size/")
	if err != nil {
		t.Fatal(err)
	}

	f := mbtest.NewReportingMetricSetV2(t, getConfig())
	defer deleteBucket(t, f)

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
		"module":   system.ModuleName,
		"datasets": []string{"package"},
	}
}

func TestPackageV1GobDecode(t *testing.T) {
	pkg := packageV1{
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

	const gobTestFile = "testdata/package.v1.gob"

	// Validate that we get the same result when decoding an earlier saved version.
	// This detects breakages to the struct or to the encoding/decoding pkgs.
	t.Run("decode_from_file", func(t *testing.T) {
		contents, err := os.ReadFile(gobTestFile)
		if err != nil {
			t.Fatal(err)
		}

		var pkgDecoded packageV1
		if err := gob.NewDecoder(bytes.NewReader(contents)).Decode(&pkgDecoded); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, pkg, pkgDecoded)
	})
}

func TestPackageDatabaseMigration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "beat.db")
	if err := copyFile("testdata/package.v1.db", dbPath); err != nil {
		t.Fatal(err)
	}

	ds := datastore.New(dbPath, 0o600)
	if err := ds.Update(migrateDatastoreSchema); err != nil {
		t.Fatal(err)
	}

	bucket, err := ds.OpenBucket(bucketNameV2)
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()

	ts, err := loadStateTimestamp(bucket)
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, 1680274980069542000, ts.UnixNano())

	pkgs, err := loadPackages(bucket)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, pkgs, 149)

	for _, pkg := range pkgs {
		if pkg.Name == "yq" {
			assert.Equal(t, &Package{
				Name:        "yq",
				Version:     "4.30.8",
				InstallTime: time.Date(2023, time.January, 17, 20, 15, 42, 0, time.UTC),
				Summary:     "Process YAML, JSON, XML, CSV and properties documents from the CLI",
				URL:         "https://github.com/mikefarah/yq",
				Type:        "brew",
			}, pkg)
		}
	}
}

func copyFile(old, new string) error {
	o, err := os.Open(old)
	if err != nil {
		return err
	}
	defer o.Close()
	n, err := os.Create(new)
	if err != nil {
		return err
	}
	defer n.Close()
	_, err = io.Copy(n, o)
	return err
}

// deleteBucket deletes the bucket from the datastore. This is workaround
// for the lack of proper test isolation. Tests may be sharing the same
// global datastore that is stored in the path.data directory and this
// prevents test side effects. The tests should be refactored to support
// true isolation with different data stores in different temp dirs.
func deleteBucket(t *testing.T, metricSet mb.ReportingMetricSetV2) {
	if err := metricSet.(*MetricSet).bucket.DeleteBucket(); err != nil {
		t.Fatal(err)
	}
}
