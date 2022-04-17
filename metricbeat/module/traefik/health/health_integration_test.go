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

//go:build integration
// +build integration

package health

import (
	"net/http"
	"testing"

	"github.com/menderesk/beats/v7/libbeat/tests/compose"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/metricbeat/module/traefik/mtest"

	"github.com/stretchr/testify/assert"
)

func makeBadRequest(host string) error {
	resp, err := http.Get("http://" + host + "/foobar")
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "traefik")

	makeBadRequest(service.Host())

	config := mtest.GetConfig("health", service.Host())
	ms := mbtest.NewReportingMetricSetV2Error(t, config)
	reporter := &mbtest.CapturingReporterV2{}

	ms.Fetch(reporter)
	assert.Nil(t, reporter.GetErrors(), "Errors while fetching metrics")

	event := reporter.GetEvents()[0]
	assert.NotNil(t, event)
	t.Logf("%s/%s event: %+v", ms.Module().Name(), ms.Name(), event)

	responseCount, _ := event.MetricSetFields.GetValue("response.count")
	assert.True(t, responseCount.(int64) >= 1)

	badResponseCount, _ := event.MetricSetFields.GetValue("response.status_codes.404")
	assert.True(t, badResponseCount.(float64) >= 1)
}
