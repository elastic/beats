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

package settings

import (
	"fmt"

	"github.com/menderesk/beats/v7/libbeat/common/productorigin"
	"github.com/menderesk/beats/v7/metricbeat/helper"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
	"github.com/menderesk/beats/v7/metricbeat/module/kibana"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet(kibana.ModuleName, "settings", New,
		mb.WithHostParser(hostParser),
	)
}

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: "http",
		DefaultPath:   kibana.SettingsPath,
		QueryParams:   "extended=true", // make Kibana fetch the cluster_uuid
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	settingsHTTP *helper.HTTP
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(r mb.ReporterV2) (err error) {
	if err = m.init(); err != nil {
		return
	}

	content, err := m.settingsHTTP.FetchContent()
	if err != nil {
		return
	}

	return eventMapping(r, content)
}

func (m *MetricSet) init() (err error) {
	httpHelper, err := helper.NewHTTP(m.BaseMetricSet)
	if err != nil {
		return err
	}

	httpHelper.SetHeaderDefault(productorigin.Header, productorigin.Beats)

	kibanaVersion, err := kibana.GetVersion(httpHelper, kibana.SettingsPath)
	if err != nil {
		return err
	}

	isSettingsAPIAvailable := kibana.IsSettingsAPIAvailable(kibanaVersion)
	if !isSettingsAPIAvailable {
		const errorMsg = "the %v metricset is only supported with Kibana >= %v. You are currently running Kibana %v"
		return fmt.Errorf(errorMsg, m.FullyQualifiedName(), kibana.SettingsAPIAvailableVersion, kibanaVersion)
	}

	m.settingsHTTP, err = helper.NewHTTP(m.BaseMetricSet)

	return
}
