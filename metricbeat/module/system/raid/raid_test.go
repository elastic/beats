// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package raid

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/system"
)

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestFetch(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("system", "raid").Fields.StringToPrint())
}

func TestFetchNoRAID(t *testing.T) {
	// Ensure that we return partial metrics when no RAID devices are present.
	tmpDir := t.TempDir()
	assert.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "sys/block"), 0755))
	c := getConfig()
	c["hostfs"] = tmpDir

	f := mbtest.NewReportingMetricSetV2Error(t, c)
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Len(t, errs, 1)
	assert.ErrorAs(t, errors.Join(errs...), &mb.PartialMetricsError{})
	assert.Contains(t, errors.Join(errs...).Error(), "failed to list RAID devices: no RAID devices found. You have probably enabled the RAID metrics on a non-RAID system.")
	assert.Empty(t, events)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"raid"},
		"hostfs":     "./_meta/testdata",
	}
}
