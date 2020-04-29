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

package node

import (
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/logstash"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet(logstash.ModuleName, "node", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

const (
	nodePath = "/_node"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: "http",
		PathConfigKey: "path",
		DefaultPath:   nodePath,
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*logstash.MetricSet
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := logstash.NewMetricSet(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		ms,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	if !m.MetricSet.XPack {
		content, err := m.HTTP.FetchContent()
		if err != nil {
			return err
		}

		return eventMapping(r, content)
	}

	pipelinesContent, overrideClusterUUID, err := logstash.GetPipelines(m.MetricSet)
	if err != nil {
		m.Logger().Error(err)
		return nil
	}

	err = eventMappingXPack(r, m, pipelinesContent, overrideClusterUUID)
	if err != nil {
		m.Logger().Error(err)
	}

	return nil
}

func (m *MetricSet) init() error {
	if m.XPack {
		return m.CheckPipelineGraphAPIsAvailable()
	}

	return nil
}
