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

package beat_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/tests/compose"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/metricbeat/module/beat"
	_ "github.com/menderesk/beats/v7/metricbeat/module/beat/state"
	_ "github.com/menderesk/beats/v7/metricbeat/module/beat/stats"
)

var metricSets = []string{
	"stats",
	"state",
}

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "metricbeat")

	for _, metricSet := range metricSets {
		f := mbtest.NewReportingMetricSetV2Error(t, beat.GetConfig(metricSet, service.Host()))
		events, errs := mbtest.ReportingFetchV2Error(f)

		require.Empty(t, errs)
		require.NotEmpty(t, events)

		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
			events[0].BeatEvent("beat", metricSet).Fields.StringToPrint())
	}
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "metricbeat")

	for _, metricSet := range metricSets {
		f := mbtest.NewReportingMetricSetV2Error(t, beat.GetConfig(metricSet, service.Host()))
		err := mbtest.WriteEventsReporterV2Error(f, t, metricSet)
		require.NoError(t, err)
	}
}
