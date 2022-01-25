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
	"strings"
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

const (
	statsPath    = "api/stats"
	settingsPath = "api/settings"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: "http",
		DefaultPath:   statsPath,
		QueryParams:   "extended=true", // make Kibana fetch the cluster_uuid
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*kibana.MetricSet
	statsHTTP         *helper.HTTP
	settingsHTTP      *helper.HTTP
	isUsageExcludable bool
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := kibana.NewMetricSet(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		MetricSet: ms,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	err := m.init()
	if err != nil {
		if m.XPackEnabled {
			m.Logger().Error(err)
			return nil
		}
		return err
	}

	now := time.Now()

	err = m.fetchStats(r, now)
	if err != nil {
		if m.XPackEnabled {
			m.Logger().Error(err)
			return nil
		}
		return err
	}

	if m.XPackEnabled {
		m.fetchSettings(r, now)
	}

	return nil
}

func (m *MetricSet) init() error {
	statsHTTP, err := helper.NewHTTP(m.BaseMetricSet)
	if err != nil {
		return err
	}

<<<<<<< HEAD
	kibanaVersion, err := kibana.GetVersion(statsHTTP, statsPath)
=======
	statsHTTP.SetHeaderDefault(productorigin.Header, productorigin.Beats)

	kibanaVersion, err := kibana.GetVersion(statsHTTP, kibana.StatsPath)
>>>>>>> 5f3dd3e39d (Add the Elastic product origin header when talking to Elasticsearch or Kibana. (#29966))
	if err != nil {
		return err
	}

	isStatsAPIAvailable := kibana.IsStatsAPIAvailable(kibanaVersion)
	if !isStatsAPIAvailable {
		const errorMsg = "the %v metricset is only supported with Kibana >= %v. You are currently running Kibana %v"
		return fmt.Errorf(errorMsg, m.FullyQualifiedName(), kibana.StatsAPIAvailableVersion, kibanaVersion)
	}
	if m.XPackEnabled {
		// Use legacy API response so we can passthru usage as-is
		statsHTTP.SetURI(statsHTTP.GetURI() + "&legacy=true")
	}

	var settingsHTTP *helper.HTTP
	if m.XPackEnabled {
		isSettingsAPIAvailable := kibana.IsSettingsAPIAvailable(kibanaVersion)
		if !isSettingsAPIAvailable {
			const errorMsg = "the %v metricset with X-Pack enabled is only supported with Kibana >= %v. You are currently running Kibana %v"
			return fmt.Errorf(errorMsg, m.FullyQualifiedName(), kibana.SettingsAPIAvailableVersion, kibanaVersion)
		}

		settingsHTTP, err = helper.NewHTTP(m.BaseMetricSet)
		if err != nil {
			return err
		}

		// HACK! We need to do this because there might be a basepath involved, so we
		// only search/replace the actual API paths
		settingsURI := strings.Replace(statsHTTP.GetURI(), statsPath, settingsPath, 1)
		settingsHTTP.SetURI(settingsURI)
	}

	m.statsHTTP = statsHTTP
	m.settingsHTTP = settingsHTTP
	m.isUsageExcludable = kibana.IsUsageExcludable(kibanaVersion)

	return nil
}

func (m *MetricSet) fetchStats(r mb.ReporterV2, now time.Time) error {

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

	if m.XPackEnabled {
		intervalMs := m.calculateIntervalMs()
		err = eventMappingStatsXPack(r, intervalMs, now, content)
		if err != nil {
			// Since this is an x-pack code path, we log the error but don't
			// return it. Otherwise it would get reported into `metricbeat-*`
			// indices.
			m.Logger().Error(err)
			return nil
		}
	} else {
		return eventMapping(r, content)
	}

	return nil
}

func (m *MetricSet) fetchSettings(r mb.ReporterV2, now time.Time) {
	content, err := m.settingsHTTP.FetchContent()
	if err != nil {
		m.Logger().Error(err)
		return
	}

	intervalMs := m.calculateIntervalMs()
	err = eventMappingSettingsXPack(r, intervalMs, now, content)
	if err != nil {
		m.Logger().Error(err)
		return
	}
}

func (m *MetricSet) calculateIntervalMs() int64 {
	return m.Module().Config().Period.Nanoseconds() / 1000 / 1000
}
