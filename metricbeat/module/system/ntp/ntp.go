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

package ntp

import (
	"fmt"

	"github.com/beevik/ntp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// MetricSet holds any configuration or state for the metricset
// Ensure MetricSet implements mb.MetricSet and mb.ReportingMetricSetV2Error
var (
	_ mb.MetricSet                 = (*MetricSet)(nil)
	_ mb.ReportingMetricSetV2Error = (*MetricSet)(nil)
)

type MetricSet struct {
	mb.BaseMetricSet
	config config
}

func init() {
	mb.Registry.MustAddMetricSet("system", "ntp", New, mb.DefaultMetricSet())
}

// New creates a new instance of the MetricSet (used for both production and test construction)
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfg := defaultConfig()
	if err := base.Module().UnpackConfig(&cfg); err != nil {
		return nil, err
	}
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}
	return &MetricSet{BaseMetricSet: base, config: cfg}, nil
}

// Fetch fetches the offset from the configured NTP server
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	response, err := ntpQueryWithOptions(m.config.Host, ntp.QueryOptions{
		Timeout: m.config.Timeout,
		Version: m.config.Version,
	})

	if err != nil {
		err := fmt.Errorf("error querying NTP server %s: %w", m.config.Host, err)
		reporter.Error(err)
		return err
	}

	reporter.Event(mb.Event{MetricSetFields: mapstr.M{
		"host":   m.config.Host,
		"offset": response.ClockOffset.Seconds(),
	}})

	return nil
}
