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

package ingest_pipeline

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

func TestMapper(t *testing.T) {
	reporter := &mbtest.CapturingReporterV2{}

	info := elasticsearch.Info{
		ClusterID:   "1234",
		ClusterName: "helloworld",
	}

	ingestData, err := ioutil.ReadFile("./_meta/test/stats.json")
	require.NoError(t, err)

	err = eventsMapping(reporter, info, ingestData, true, true)
	require.NoError(t, err)
	require.Equal(t, 0, len(reporter.GetErrors()))

	// 1 pipeline, 5 processors
	allEvents := reporter.GetEvents()
	var pipelineEvents []mb.Event
	var processorEvents []mb.Event

	for _, event := range allEvents {
		if val, _ := event.MetricSetFields.GetValue("processor"); val != nil {
			processorEvents = append(processorEvents, event)
		} else {
			pipelineEvents = append(pipelineEvents, event)
		}
	}

	require.Equal(t, 6, len(allEvents))

	t.Run("Test pipeline events", func(t *testing.T) {
		require.Equal(t, 1, len(pipelineEvents))
		ev := pipelineEvents[0]

		requireMetricSetFields(t, ev, "name", "pipeline1")
		requireMetricSetFields(t, ev, "total.count", 19271022)
		requireMetricSetFields(t, ev, "total.failed", 100)
		requireMetricSetFields(t, ev, "total.time.total.ms", 823888)
		requireMetricSetFields(t, ev, "total.time.self.ms", 823888-4607) // subtract out pipeline processor
	})

	t.Run("Test processor events", func(t *testing.T) {
		require.Equal(t, 5, len(processorEvents))
		ev := processorEvents[0]

		// There's no special handling for different processors, so just test the first one
		requireMetricSetFields(t, ev, "name", "pipeline1")
		requireMetricSetFields(t, ev, "processor.order_index", 0)
		requireMetricSetFields(t, ev, "processor.type", "set")
		requireMetricSetFields(t, ev, "processor.type_tag", "set:tag1")
		requireMetricSetFields(t, ev, "processor.count", 19271022)
		requireMetricSetFields(t, ev, "processor.failed", 100)
		requireMetricSetFields(t, ev, "processor.time.total.ms", 256275)
	})
}

func TestSampling(t *testing.T) {
	reporter := &mbtest.CapturingReporterV2{}

	info := elasticsearch.Info{
		ClusterID:   "1234",
		ClusterName: "helloworld",
	}

	ingestData, err := ioutil.ReadFile("./_meta/test/stats.json")
	require.NoError(t, err)

	err = eventsMapping(reporter, info, ingestData, true, false) // set sampling to false
	require.NoError(t, err)
	require.Equal(t, 0, len(reporter.GetErrors()))

	// 1 pipeline, 0 processors
	allEvents := reporter.GetEvents()
	var pipelineEvents []mb.Event
	var processorEvents []mb.Event

	for _, event := range allEvents {
		if val, _ := event.MetricSetFields.GetValue("processor"); val != nil {
			processorEvents = append(processorEvents, event)
		} else {
			pipelineEvents = append(pipelineEvents, event)
		}
	}

	require.Equal(t, 1, len(allEvents))
	require.Equal(t, 1, len(pipelineEvents))
	require.Equal(t, 0, len(processorEvents))
}

func requireMetricSetFields(t *testing.T, event mb.Event, fieldName string, expected interface{}) {
	val, err := event.MetricSetFields.GetValue(fieldName)
	require.NoError(t, err)
	require.Equal(t, expected, val)
}
