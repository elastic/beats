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
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/productorigin"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/kibana"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet(kibana.ModuleName, "stats", New,
		mb.WithHostParser(hostParser),
	)
}

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: "http",
		DefaultPath:   kibana.StatsPath,
		QueryParams:   "extended=true", // make Kibana fetch the cluster_uuid
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*kibana.MetricSet
	statsHTTP                         *helper.HTTP
	isUsageExcludable                 bool
	lastRunningKibanaMessageTimestamp time.Time
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := kibana.NewMetricSet(base)
	if err != nil {
		return nil, err
	}

	statsHTTP, err := helper.NewHTTP(ms.BaseMetricSet)
	if err != nil {
		return nil, err
	}

	statsHTTP.SetHeaderDefault(productorigin.Header, productorigin.Beats)

	return &MetricSet{
		MetricSet: ms,
		statsHTTP: statsHTTP,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(r mb.ReporterV2) (err error) {
	err, versionSupported := m.init()
	if err != nil {
		return err
	}

	if !versionSupported {
		return nil
	}

	if err = m.fetchStats(r); err != nil {
		return fmt.Errorf("error trying to get stats data from Kibana: %w", err)
	}

	return nil
}

func (m *MetricSet) init() (err error, versionSupported bool) {
	kibanaVersion, err := kibana.GetVersion(m.statsHTTP, kibana.StatsPath)
	if err != nil {
		return err, false
	}

	isStatsAPIAvailable := kibana.IsStatsAPIAvailable(kibanaVersion)
	if !isStatsAPIAvailable {
		if time.Since(m.lastRunningKibanaMessageTimestamp) > 5*time.Minute {
			m.lastRunningKibanaMessageTimestamp = time.Now()
			const errorMsg = "the %v metricset is only supported with Kibana >= %v. You are currently running Kibana %v"
			m.Logger().Debugf(errorMsg, m.FullyQualifiedName(), kibana.ActionsAPIAvailableVersion, kibanaVersion)
		}

		return nil, false
	}

	m.isUsageExcludable = kibana.IsUsageExcludable(kibanaVersion)

	return err, true
}

func (m *MetricSet) fetchStats(r mb.ReporterV2) error {
	var content []byte
	var err error

	// Add exclude_usage=true if the Kibana Version supports it
	if m.isUsageExcludable {
		origURI := m.statsHTTP.GetURI()
		defer m.statsHTTP.SetURI(origURI)

		m.statsHTTP.SetURI(origURI + "&exclude_usage=true")
	}

	content, err = m.statsHTTP.FetchContent()
	if err != nil {
		return err
	}

	return eventMapping(r, content, m.XPackEnabled)
}
