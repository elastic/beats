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

package index_summary

import (
	"fmt"
	"net/url"

	"github.com/elastic/beats/v9/metricbeat/mb"
	"github.com/elastic/beats/v9/metricbeat/mb/parse"
	"github.com/elastic/beats/v9/metricbeat/module/elasticsearch"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet(elasticsearch.ModuleName, "index_summary", New,
		mb.WithHostParser(hostParser),
		mb.WithNamespace("elasticsearch.index.summary"),
	)
}

const (
	nodeStatsPath = "/_nodes/stats"

	nodeStatsParameters = "level=node&filter_path=nodes.*.indices.docs,nodes.*.indices.indexing.index_total,nodes.*.indices.indexing.index_time_in_millis,nodes.*.indices.search.query_total,nodes.*.indices.search.query_time_in_millis,nodes.*.indices.segments.count,nodes.*.indices.segments.memory_in_bytes,nodes.*.indices.store.size_in_bytes,nodes.*.indices.store.total_data_set_size_in_bytes"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: "http",
		PathConfigKey: "path",
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*elasticsearch.MetricSet
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Get the stats from the local node
	ms, err := elasticsearch.NewMetricSet(base, nodeStatsPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch gathers stats for each index from the _stats API
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	shouldSkip, err := m.ShouldSkipFetch()
	if err != nil {
		return err
	}
	if shouldSkip {
		return nil
	}

	info, err := elasticsearch.GetInfo(m.HTTP, m.HostData().SanitizedURI+nodeStatsPath)
	if err != nil {
		return fmt.Errorf("failed to get info from Elasticsearch: %w", err)
	}

	if err := m.updateServicePath(); err != nil {
		return err
	}

	content, err := m.FetchContent()
	if err != nil {
		return err
	}

	return eventMapping(r, info, content, m.XPackEnabled)
}

func (m *MetricSet) updateServicePath() error {
	p, err := getServicePath()
	if err != nil {
		return err
	}

	m.SetServiceURI(p)
	return nil
}

func getServicePath() (string, error) {
	currPath := nodeStatsPath
	u, err := url.Parse(currPath)
	if err != nil {
		return "", err
	}

	u.RawQuery += nodeStatsParameters

	return u.String(), nil
}
