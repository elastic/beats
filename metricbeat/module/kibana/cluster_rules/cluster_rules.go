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

package cluster_rules

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/kibana"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet(kibana.ModuleName, "cluster_rules", New,
		mb.WithHostParser(hostParser),
	)
}

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: "http",
		DefaultPath:   kibana.ClusterRulesPath,
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*kibana.MetricSet
	rulesHTTP *helper.HTTP
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := kibana.NewMetricSet(base)
	if err != nil {
		return nil, err
	}

	rulesHTTP, err := helper.NewHTTP(ms.BaseMetricSet)
	if err != nil {
		return nil, err
	}

	kibanaVersion, err := kibana.GetVersion(rulesHTTP, kibana.ClusterRulesPath)
	if err != nil {
		return nil, err
	}

	isMetricsAPIAvailable := kibana.IsRulesAPIAvailable(kibanaVersion)
	if !isMetricsAPIAvailable {
		const errorMsg = "the %v cluster rules is only supported with Kibana >= %v. You are currently running Kibana %v"
		return nil, fmt.Errorf(errorMsg, ms.FullyQualifiedName(), kibana.RulesAPIAvailableVersion, kibanaVersion)
	}

	return &MetricSet{
		MetricSet: ms,
		rulesHTTP: rulesHTTP,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(r mb.ReporterV2) (err error) {
	if err = m.fetchMetrics(r); err != nil {
		return fmt.Errorf("error trying to get cluster rule data from Kibana: %w", err)
	}

	return nil
}

func (m *MetricSet) fetchMetrics(r mb.ReporterV2) error {
	var content []byte
	var err error

	content, err = m.rulesHTTP.FetchContent()
	if err != nil {
		return err
	}

	return eventMapping(r, content, m.XPackEnabled)
}
