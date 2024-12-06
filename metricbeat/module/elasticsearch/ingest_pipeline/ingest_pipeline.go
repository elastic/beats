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
	"fmt"
	"math"
	"net/url"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(elasticsearch.ModuleName, "ingest_pipeline", New,
		mb.WithHostParser(elasticsearch.HostParser),
		mb.DefaultMetricSet(),
	)
}

const (
	statsPathCluster = "/_nodes/stats/ingest"
	statsPathNode    = "/_nodes/_local/stats/ingest"
)

// IngestMetricSet type defines all fields of the IngestMetricSet
type IngestMetricSet struct {
	*elasticsearch.MetricSet

	// fetchCounter counts the number of times the Fetch method has been called.
	// Used for sampling
	fetchCounter int

	// Rate at which processor level events should be sampled
	sampleProcessorsEveryN int
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The elasticsearch ingest metricset is beta.")
	ms, err := elasticsearch.NewMetricSet(base, statsPathCluster)
	if err != nil {
		return nil, err
	}

	config := struct {
		ProcessorSampleRate float64 `config:"ingest_pipeline.processor_sample_rate"`
	}{
		ProcessorSampleRate: 0.25,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	var sampleProcessorsEveryN int
	if config.ProcessorSampleRate == 0 {
		sampleProcessorsEveryN = 0
	} else {
		sampleProcessorsEveryN = int(math.Round(1.0 / math.Min(1.0, config.ProcessorSampleRate)))
	}

	base.Logger().Debugf("Sampling ingest_pipeline processor stats every %d fetches", sampleProcessorsEveryN)

	return &IngestMetricSet{
		MetricSet:              ms,
		fetchCounter:           0,
		sampleProcessorsEveryN: sampleProcessorsEveryN,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *IngestMetricSet) Fetch(report mb.ReporterV2) error {
	uri, err := url.Parse(m.GetURI())
	if err != nil {
		return err
	}

	if m.Scope == elasticsearch.ScopeCluster {
		uri.Path = statsPathCluster
	} else {
		uri.Path = statsPathNode
	}
	m.HTTP.SetURI(uri.String())

	content, err := m.HTTP.FetchContent()
	if err != nil {
		return err
	}

	info, err := elasticsearch.GetInfo(m.HTTP, m.HostData().SanitizedURI)
	if err != nil {
		return fmt.Errorf("failed to get info from Elasticsearch: %w", err)
	}

	m.fetchCounter++ // It's fine if this overflows, it's only used for modulo
	sampleProcessors := m.fetchCounter%m.sampleProcessorsEveryN == 0
	m.Logger().Debugf("Sampling ingest_pipeline processor stats: %v", sampleProcessors)
	return eventsMapping(report, info, content, m.XPackEnabled, sampleProcessors)
}
