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

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"

	"github.com/elastic/elastic-agent-libs/version"
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
	statsPath = "/_stats"

	allowClosedIndices = "forbid_closed_indices=false"
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
	ms, err := elasticsearch.NewMetricSet(base, statsPath)
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

	info, err := elasticsearch.GetInfo(m.HTTP, m.HostData().SanitizedURI+statsPath)
	if err != nil {
		return fmt.Errorf("failed to get info from Elasticsearch: %w", err)
	}

	if err := m.updateServicePath(*info.Version.Number); err != nil {
		return err
	}

	content, err := m.HTTP.FetchContent()
	if err != nil {
		return err
	}

	return eventMapping(r, info, content, m.XPackEnabled)
}

func (m *MetricSet) updateServicePath(esVersion version.V) error {
	p, err := getServicePath(esVersion)
	if err != nil {
		return err
	}

	m.SetServiceURI(p)
	return nil
}

func getServicePath(esVersion version.V) (string, error) {
	currPath := statsPath
	u, err := url.Parse(currPath)
	if err != nil {
		return "", err
	}

	if !esVersion.LessThan(elasticsearch.BulkStatsAvailableVersion) {
		u.RawQuery += allowClosedIndices
	}

	return u.String(), nil
}
