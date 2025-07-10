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

package node_stats

import (
	"net/url"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet(elasticsearch.ModuleName, "node_stats", New,
		mb.WithHostParser(elasticsearch.HostParser),
		mb.DefaultMetricSet(),
		mb.WithNamespace("elasticsearch.node.stats"),
	)
}

const (
	statsMetrics       = "jvm,indices,fs,os,process,transport,thread_pool,indexing_pressure,ingest"
	indexMetrics       = "bulk,docs,get,merge,translog,fielddata,indexing,query_cache,request_cache,search,shard_stats,store,segments,refresh,flush"
	nodeLocalStatsPath = "/_nodes/_local/stats/" + statsMetrics + "/" + indexMetrics
	nodesAllStatsPath  = "/_nodes/_all/stats/" + statsMetrics + "/" + indexMetrics
	// versions < 8
	legacyLocalStatsPath = "/_nodes/_local/stats"
	legacyAllStatsPath   = "/_nodes/_all/stats"
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*elasticsearch.MetricSet
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Get the stats from the local node
	ms, err := elasticsearch.NewMetricSet(base, "") // servicePath will be set in Fetch()
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	info, err := elasticsearch.GetInfo(m.HTTP, m.GetServiceURI())
	if err != nil {
		return err
	}

	if err := m.updateServiceURI(info.Version.Number.Major); err != nil {
		return err
	}

	content, err := m.FetchContent()
	if err != nil {
		return err
	}

	return eventsMapping(r, m.MetricSet, info, content, m.XPackEnabled)
}

func (m *MetricSet) updateServiceURI(majorVersion int) error {
	u, err := getServiceURI(m.GetURI(), m.Scope, majorVersion)
	if err != nil {
		return err
	}

	m.SetURI(u)
	return nil

}

func getServiceURI(currURI string, scope elasticsearch.Scope, majorVersion int) (string, error) {
	u, err := url.Parse(currURI)
	if err != nil {
		return "", err
	}

	if majorVersion >= 8 {
		if scope == elasticsearch.ScopeCluster {
			u.Path = nodesAllStatsPath
		} else {
			u.Path = nodeLocalStatsPath
		}
	} else {
		if scope == elasticsearch.ScopeCluster {
			u.Path = legacyAllStatsPath
		} else {
			u.Path = legacyLocalStatsPath
		}
	}

	return u.String(), nil
}
