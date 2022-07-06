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

package mgr_cluster_health

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/ceph/mgr"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	defaultScheme      = "https"
	defaultPath        = "/request"
	defaultQueryParams = "wait=1"

	cephStatusPrefix         = "status"
	cephTimeSyncStatusPrefix = "time-sync-status"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		QueryParams:   defaultQueryParams,
	}.Build()
)

func init() {
	mb.Registry.MustAddMetricSet("ceph", "mgr_cluster_health", New,
		mb.WithHostParser(hostParser),
	)
}

type MetricSet struct {
	*mgr.MetricSet
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	metricSet, err := mgr.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{metricSet}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	m.HTTP.SetBody([]byte(fmt.Sprintf(`{"prefix": "%s", "format": "json"}`, cephStatusPrefix)))
	statusContent, err := m.HTTP.FetchContent()
	if err != nil {
		return err
	}

	m.HTTP.SetBody([]byte(fmt.Sprintf(`{"prefix": "%s", "format": "json"}`, cephTimeSyncStatusPrefix)))
	timeStatusContent, err := m.HTTP.FetchContent()
	if err != nil {
		return err
	}

	event, err := eventMapping(statusContent, timeStatusContent)
	if err != nil {
		return err
	}

	reporter.Event(mb.Event{
		ModuleFields: mapstr.M{
			"cluster_health": event,
		}})
	return nil
}
