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
	"strings"
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v9/metricbeat/mb/testing"
)

type ntpSuccess struct{}

func (n *ntpSuccess) query(host string, opt ntp.QueryOptions) (*ntp.Response, error) {
	return &ntp.Response{ClockOffset: time.Duration(1.23 * float64(time.Second))}, nil
}

func (n *ntpSuccess) validate(_ *ntp.Response) error {
	return nil
}

func getTestConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":      "system",
		"metricsets":  []string{"ntp"},
		"ntp.servers": []string{"0.time.tom.com", "1.time.tom.com"},
	}
}

func TestFetchOffset_Success(t *testing.T) {
	metricSet := mbtest.NewReportingMetricSetV2Error(t, getTestConfig())
	ntpMetricSet, ok := metricSet.(*MetricSet)
	require.True(t, ok, "metricSet is not of type *MetricSet")

	ntpMetricSet.queryProvider = &ntpSuccess{}

	events, errs := mbtest.ReportingFetchV2Error(ntpMetricSet)
	require.Empty(t, errs, "expected no errors, got: %v", errs)

	assert.Len(t, events, 2, "expected 2 events")
	for _, event := range events {
		msFields := event.MetricSetFields
		offset, ok := msFields["offset"]
		assert.True(t, ok, "offset not found in event")
		assert.Equal(t, int64(1230000000), offset, "offset should be 1230000000 nanoseconds")

		host, ok := msFields["host"]
		assert.True(t, ok, "host not found in event")
		hostStr, _ := host.(string)
		assert.True(t, strings.HasSuffix(hostStr, "time.tom.com"), "host should match configured host")
	}
}

type ntpError struct{}

func (n *ntpError) query(host string, opt ntp.QueryOptions) (*ntp.Response, error) {
	return nil, errors.New("ntp error")
}

func (n *ntpError) validate(_ *ntp.Response) error {
	return nil
}

func TestFetchOffset_Error(t *testing.T) {
	metricSet := mbtest.NewReportingMetricSetV2Error(t, getTestConfig())
	ntpMetricSet, ok := metricSet.(*MetricSet)
	require.True(t, ok, "metricSet is not of type *MetricSet")

	ntpMetricSet.queryProvider = &ntpError{}

	_, errs := mbtest.ReportingFetchV2Error(ntpMetricSet)
	assert.NotEmpty(t, errs, "expected error, got none")
}

type ntpValidationFailed struct {
	Called int
}

func (n *ntpValidationFailed) query(host string, opt ntp.QueryOptions) (*ntp.Response, error) {
	return &ntp.Response{ClockOffset: time.Duration(1.23 * float64(time.Second))}, nil
}

func (n *ntpValidationFailed) validate(_ *ntp.Response) error {
	n.Called++
	return errors.New("ntp validation error")
}

func TestFetchOffset_ValidationError(t *testing.T) {
	t.Run("ntp.validate=true", func(t *testing.T) {
		metricSet := mbtest.NewReportingMetricSetV2Error(t, getTestConfig())
		ntpMetricSet, ok := metricSet.(*MetricSet)
		require.True(t, ok, "metricSet is not of type *MetricSet")

		queryProvider := &ntpValidationFailed{}

		ntpMetricSet.queryProvider = queryProvider

		events, errs := mbtest.ReportingFetchV2Error(ntpMetricSet)
		assert.Equal(t, 2, queryProvider.Called, "expected validate to be called twice, called %d times", queryProvider.Called)
		require.Empty(t, events, "expected no events, got %v", events)
		require.Empty(t, errs, "expected no errors, got: %v", errs)
	})

	t.Run("ntp.validate=false", func(t *testing.T) {
		config := getTestConfig()
		config["ntp.validate"] = "false"

		metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
		ntpMetricSet, ok := metricSet.(*MetricSet)
		require.True(t, ok, "metricSet is not of type *MetricSet")

		queryProvider := &ntpValidationFailed{}

		ntpMetricSet.queryProvider = queryProvider

		_, errs := mbtest.ReportingFetchV2Error(ntpMetricSet)
		assert.Equal(t, 0, queryProvider.Called, "expected validate to be NOT called, called %d times", queryProvider.Called)
		require.Empty(t, errs, "expected no errors, got: %v", errs)
	})

}
