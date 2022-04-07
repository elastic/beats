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

package ksm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v8/metricbeat/module/linux"
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

	testKSM := ksmData{
		PagesShared:      100,
		PagesSharing:     10,
		PagesUnshared:    0,
		PagesVolatile:    0,
		FullScans:        2000,
		StableNodeChains: 0,
		StableNodeDups:   0,
	}

	rawEvent := events[0].BeatEvent("linux", "ksm").Fields["linux"].(common.MapStr)["ksm"].(common.MapStr)["stats"]

	assert.Equal(t, testKSM, rawEvent)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "linux",
		"metricsets": []string{"ksm"},
		"hostfs":     "./_meta/testdata",
	}
}
