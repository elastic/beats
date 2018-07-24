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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/kibana"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("kibana", "stats", New,
		mb.WithHostParser(hostParser),
	)
}

const (
	statsPath                      = "api/stats"
	kibanaStatsAPIAvailableVersion = "6.4.0"
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
	mb.BaseMetricSet
	http         *helper.HTTP
	xPackEnabled bool
}

func isKibanaStatsAPIAvailable(kibanaVersion string) (bool, error) {
	currentVersion, err := common.NewVersion(kibanaVersion)
	if err != nil {
		return false, err
	}

	wantVersion, err := common.NewVersion(kibanaStatsAPIAvailableVersion)
	if err != nil {
		return false, err
	}

	return !currentVersion.LessThan(wantVersion), nil
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The kibana stats metricset is experimental")

	config := kibana.DefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.XPackEnabled {
		cfgwarn.Experimental("The experimental xpack.enabled flag in kibana/stats metricset is enabled.")
	}

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	kibanaVersion, err := kibana.GetVersion(http, statsPath)
	if err != nil {
		return nil, err
	}

	isAPIAvailable, err := isKibanaStatsAPIAvailable(kibanaVersion)
	if err != nil {
		return nil, err
	}

	if !isAPIAvailable {
		const errorMsg = "The kibana stats metricset is only supported with Kibana >= %v. You are currently running Kibana %v"
		return nil, fmt.Errorf(errorMsg, kibanaStatsAPIAvailableVersion, kibanaVersion)
	}

	return &MetricSet{
		base,
		http,
		config.XPackEnabled,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(r mb.ReporterV2) {
	content, err := m.http.FetchContent()
	if err != nil {
		r.Error(err)
		return
	}

	if m.xPackEnabled {
		intervalMs := m.Module().Config().Period.Nanoseconds() / 1000 / 1000
		eventMappingXPack(r, intervalMs, content)
	} else {
		eventMapping(r, content)
	}

}
