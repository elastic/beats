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
	"errors"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/beevik/ntp"
)

// MetricSet holds any configuration or state for the metricset
// Ensure MetricSet implements mb.MetricSet and mb.ReportingMetricSetV2Error
var (
	_ mb.MetricSet                 = (*MetricSet)(nil)
	_ mb.ReportingMetricSetV2Error = (*MetricSet)(nil)
)

type ntpQueryProvider interface {
	query(host string, options ntp.QueryOptions) (*ntp.Response, error)
}

type beevikNTPQueryProvider struct{}

func (n *beevikNTPQueryProvider) query(host string, options ntp.QueryOptions) (*ntp.Response, error) {
	response, err := ntp.QueryWithOptions(host, options)
	if err != nil {
		return nil, err
	}

	if err := response.Validate(); err != nil {
		return nil, err
	}

	return response, nil
}

type MetricSet struct {
	mb.BaseMetricSet
	config        config
	queryProvider ntpQueryProvider
}

func init() {
	mb.Registry.MustAddMetricSet("system", "ntp", New)
}

// New creates a new instance of the NTP MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfg := defaultConfig()
	if err := base.Module().UnpackConfig(&cfg); err != nil {
		return nil, err
	}
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}
	return &MetricSet{BaseMetricSet: base, config: cfg, queryProvider: &beevikNTPQueryProvider{}}, nil
}

// Fetch fetches the offset from the configured NTP server
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	var wg sync.WaitGroup
	fetchErrors := make(chan error, len(m.config.Servers))
	wg.Add(len(m.config.Servers))

	for _, server := range m.config.Servers {
		go func() {
			defer wg.Done()

			response, err := m.queryProvider.query(server, ntp.QueryOptions{
				Timeout: m.config.Timeout,
				Version: m.config.Version,
			})
			if err != nil {
				err := fmt.Errorf("error querying NTP server %s: %w", server, err)
				reporter.Error(err)
				fetchErrors <- err
				return
			}

			reporter.Event(mb.Event{MetricSetFields: mapstr.M{
				"host":   server,
				"offset": response.ClockOffset.Nanoseconds(),
			}})
		}()
	}

	wg.Wait()
	close(fetchErrors)

	var errs []error
	for err := range fetchErrors {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
