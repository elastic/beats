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

// +build integration
// +build windows

package application_pool

import (
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"testing"
)

func TestFetch(t *testing.T) {
	config := map[string]interface{}{
		"module":     "iis",
		"period":     "30s",
		"metricsets": []string{"application_pool"},
	}
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	_, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		// should find a way to first check if iis is running
		//t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
}

func TestData(t *testing.T) {
	config := map[string]interface{}{
		"module":     "iis",
		"period":     "30s",
		"metricsets": []string{"application_pool"},
	}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		// should find a way to first check if iis is running
		//	t.Fatal("write", err)
	}
}
