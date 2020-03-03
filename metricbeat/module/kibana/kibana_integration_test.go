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

package kibana_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/kibana"
	_ "github.com/elastic/beats/v7/metricbeat/module/kibana/stats"
)

var xpackMetricSets = []string{
	"stats",
}

func TestXPackEnabled(t *testing.T) {
	service := compose.EnsureUpWithTimeout(t, 300, "kibana")

	metricSetToTypeMap := map[string]string{
		"stats": "kibana_stats",
	}

	config := getXPackConfig(service.Host())

	metricSets := mbtest.NewReportingMetricSetV2Errors(t, config)
	for _, metricSet := range metricSets {
		events, errs := mbtest.ReportingFetchV2Error(metricSet)
		require.Empty(t, errs)
		require.NotEmpty(t, events)

		event := events[0]
		require.Equal(t, metricSetToTypeMap[metricSet.Name()], event.RootFields["type"])
		require.Regexp(t, `^.monitoring-kibana-\d-mb`, event.Index)
	}
}

func getXPackConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":        kibana.ModuleName,
		"metricsets":    xpackMetricSets,
		"hosts":         []string{host},
		"xpack.enabled": true,
	}
}
