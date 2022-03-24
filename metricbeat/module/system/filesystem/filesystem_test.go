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

//go:build darwin || freebsd || linux || openbsd || windows
// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/system"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)
	logp.DevelopmentSetup()
	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	for _, event := range events {
		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
			event.BeatEvent("system", "filesystem").Fields.StringToPrint())
	}

}

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	//ignoreTypes := append(fs.DefaultIgnoredTypes(resolve.NewTestResolver("")), "fuse.lxcfs", "fuse.gvfsd-fuse", "nsfs", "squashfs")
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"filesystem"},
		//"filesystem.ignore_types": []string{},
	}
}
