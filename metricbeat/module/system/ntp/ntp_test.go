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

package ntp

import (
	"errors"
	"testing"
	"time"

	"github.com/beevik/ntp"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"
)

func getTestConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"ntp"},
		"host":       "localhost:123",
	}
}

func TestFetchOffset_Success(t *testing.T) {
	ntpQueryOrig := ntpQueryWithOptions
	ntpQueryWithOptions = func(host string, opt ntp.QueryOptions) (*ntp.Response, error) {
		return &ntp.Response{ClockOffset: time.Duration(1.23 * float64(time.Second))}, nil
	}
	defer func() { ntpQueryWithOptions = ntpQueryOrig }()

	metricSet := mbtest.NewReportingMetricSetV2Error(t, getTestConfig())
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	assert.Empty(t, errs, "expected no errors, got: %v", errs)
	assert.Len(t, events, 1, "expected 1 event")
	msFields := events[0].MetricSetFields
	offset, ok := msFields["offset"]
	assert.True(t, ok, "offset not found in event")
	assert.Equal(t, 1.23, offset, "offset should be 1.23 seconds")

	host, ok := msFields["host"]
	assert.True(t, ok, "host not found in event")
	assert.Equal(t, "localhost:123", host, "host should match configured host")
}

func TestFetchOffset_Error(t *testing.T) {
	ntpQueryOrig := ntpQueryWithOptions
	ntpQueryWithOptions = func(host string, opt ntp.QueryOptions) (*ntp.Response, error) {
		return nil, errors.New("ntp error")
	}
	defer func() { ntpQueryWithOptions = ntpQueryOrig }()

	metricSet := mbtest.NewReportingMetricSetV2Error(t, getTestConfig())
	_, errs := mbtest.ReportingFetchV2Error(metricSet)
	assert.NotEmpty(t, errs, "expected error, got none")
}
