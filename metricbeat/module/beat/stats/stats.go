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

package stats

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
	"github.com/elastic/beats/v8/metricbeat/module/beat"
)

func init() {
	mb.Registry.MustAddMetricSet(beat.ModuleName, "stats", New,
		mb.WithHostParser(hostParser),
	)
}

const (
	statsPath = "stats"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: "http",
		DefaultPath:   statsPath,
	}.Build()
)

// MetricSet defines all fields of the MetricSet
type MetricSet struct {
	*beat.MetricSet
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := beat.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	content, err := m.HTTP.FetchContent()
	if err != nil {
		return err
	}

	info, err := beat.GetInfo(m.MetricSet)
	if err != nil {
		return err
	}

	clusterUUID, err := m.getClusterUUID()
	if err != nil {
		return err
	}

	return eventMapping(r, *info, clusterUUID, content, m.XPackEnabled)
}

func (m *MetricSet) getClusterUUID() (string, error) {
	state, err := beat.GetState(m.MetricSet)
	if err != nil {
		return "", errors.Wrap(err, "could not get state information")
	}

	clusterUUID := state.Monitoring.ClusterUUID
	if clusterUUID != "" {
		return clusterUUID, nil
	}

	if state.Output.Name != "elasticsearch" {
		return "", nil
	}

	clusterUUID = state.Outputs.Elasticsearch.ClusterUUID
	if clusterUUID == "" {
		// Output is ES but cluster UUID could not be determined. No point sending monitoring
		// data with empty cluster UUID since it will not be associated with the correct ES
		// production cluster. Log error instead.
		return "", beat.ErrClusterUUID
	}

	return clusterUUID, nil
}
