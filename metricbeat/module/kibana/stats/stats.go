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
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/kibana"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet(kibana.ModuleName, "stats", New,
		mb.WithHostParser(hostParser),
	)
}

const (
	statsPath             = "api/stats"
	settingsPath          = "api/settings"
	usageCollectionPeriod = 24 * time.Hour
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
	statsHTTP            *helper.HTTP
	settingsHTTP         *helper.HTTP
	usageLastCollectedOn time.Time
	isUsageExcludable    bool
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := kibana.NewMetricSet(base)
	if err != nil {
		return nil, err
	}

	statsHTTP, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	kibanaVersion, err := kibana.GetVersion(statsHTTP, statsPath)
	if err != nil {
		return nil, err
	}

	isStatsAPIAvailable := kibana.IsStatsAPIAvailable(kibanaVersion)
	if err != nil {
		return nil, err
	}

	if !isStatsAPIAvailable {
		const errorMsg = "The %v metricset is only supported with Kibana >= %v. You are currently running Kibana %v"
		return nil, fmt.Errorf(errorMsg, base.FullyQualifiedName(), kibana.StatsAPIAvailableVersion, kibanaVersion)
	}

	if ms.XPackEnabled {
		// Use legacy API response so we can passthru usage as-is
		statsHTTP.SetURI(statsHTTP.GetURI() + "&legacy=true")
	}

	var settingsHTTP *helper.HTTP
	if ms.XPackEnabled {
		isSettingsAPIAvailable := kibana.IsSettingsAPIAvailable(kibanaVersion)
		if err != nil {
			return nil, err
		}

		if !isSettingsAPIAvailable {
			const errorMsg = "The %v metricset with X-Pack enabled is only supported with Kibana >= %v. You are currently running Kibana %v"
			return nil, fmt.Errorf(errorMsg, ms.FullyQualifiedName(), kibana.SettingsAPIAvailableVersion, kibanaVersion)
		}

		settingsHTTP, err = helper.NewHTTP(base)
		if err != nil {
			return nil, err
		}

		// HACK! We need to do this because there might be a basepath involved, so we
		// only search/replace the actual API paths
		settingsURI := strings.Replace(statsHTTP.GetURI(), statsPath, settingsPath, 1)
		settingsHTTP.SetURI(settingsURI)
	}

	return &MetricSet{
		ms,
		statsHTTP,
		settingsHTTP,
		time.Time{},
		kibana.IsUsageExcludable(kibanaVersion),
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	now := time.Now()

	err := m.fetchStats(r, now)
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

func (m *MetricSet) fetchStats(r mb.ReporterV2, now time.Time) error {
	// Collect usage stats only once every usageCollectionPeriod
	if m.isUsageExcludable {
		origURI := m.statsHTTP.GetURI()
		defer m.statsHTTP.SetURI(origURI)

		shouldCollectUsage := m.shouldCollectUsage(now)
		if shouldCollectUsage {
			m.usageLastCollectedOn = now
		}
		m.statsHTTP.SetURI(origURI + "&exclude_usage=" + strconv.FormatBool(!shouldCollectUsage))
	}

	content, err := m.statsHTTP.FetchContent()
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

func (m *MetricSet) shouldCollectUsage(now time.Time) bool {
	return now.Sub(m.usageLastCollectedOn) > usageCollectionPeriod
}
