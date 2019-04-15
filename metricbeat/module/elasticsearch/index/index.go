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

package index

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet(elasticsearch.ModuleName, "index", New,
		mb.WithHostParser(elasticsearch.HostParser),
		mb.WithNamespace("elasticsearch.index"),
	)
}

const (
	statsMetrics = "docs,fielddata,indexing,merge,search,segments,store,refresh,query_cache,request_cache"
	statsPath    = "/_stats/" + statsMetrics
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*elasticsearch.MetricSet
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// TODO: This currently gets index data for all indices. Make it configurable.
	ms, err := elasticsearch.NewMetricSet(base, statsPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch gathers stats for each index from the _stats API
func (m *MetricSet) Fetch(r mb.ReporterV2) error {

	isMaster, err := elasticsearch.IsMaster(m.HTTP, m.HostData().SanitizedURI+statsPath)
	if err != nil {
		return errors.Wrap(err, "error determining if connected Elasticsearch node is master")
	}

	// Not master, no event sent
	if !isMaster {
		m.Logger().Debug("trying to fetch index stats from a non-master node")
		return nil
	}

	content, err := m.HTTP.FetchContent()
	if err != nil {
		return err
	}

	info, err := elasticsearch.GetInfo(m.HTTP, m.HostData().SanitizedURI)
	if err != nil {
		return errors.Wrap(err, "failed to get info from Elasticsearch")
	}

	if m.XPack {
		err = eventsMappingXPack(r, m, *info, content)
		if err != nil {
			// Since this is an x-pack code path, we log the error but don't
			// return it. Otherwise it would get reported into `metricbeat-*`
			// indices.
			m.Logger().Error(err)
			return nil
		}
	} else {
		return eventsMapping(r, *info, content)
	}

	return nil
}
