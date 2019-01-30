// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package login

import (
	"encoding/binary"
	"io/ioutil"
	"os"
	"testing"

	"github.com/elastic/beats/auditbeat/core"
	"github.com/elastic/beats/libbeat/paths"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	if byteOrder != binary.LittleEndian {
		t.Skip("Test only works on little-endian systems - skipping.")
	}

	defer setup(t)()

	f := mbtest.NewReportingMetricSetV2(t, getConfig())

	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("received error: %+v", errs[0])
	}

	if len(events) == 0 {
		t.Fatal("no events were generated")
	} else if len(events) != 1 {
		t.Fatal("only one event expected")
	}

	fullEvent := mbtest.StandardizeEvent(f, events[0], core.AddDatasetToEvent)
	mbtest.WriteEventToDataJSON(t, fullEvent, "")
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":                  "system",
		"datasets":                []string{"login"},
		"login.wtmp_file_pattern": "../../../tests/files/wtmp",
		"login.btmp_file_pattern": "",
	}
}

// setup is copied from file_integrity/metricset_test.go.
// TODO: Move to shared location and use in all unit tests.
func setup(t testing.TB) func() {
	// path.data should be set so that the DB is written to a predictable location.
	var err error
	paths.Paths.Data, err = ioutil.TempDir("", "beat-data-dir")
	if err != nil {
		t.Fatal()
	}
	return func() { os.RemoveAll(paths.Paths.Data) }
}
