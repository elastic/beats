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

// +build windows

package application_pool

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("iis", "application_pool", New, mb.DefaultMetricSet())
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	log    *logp.Logger
	reader *Reader
}

// Config for the iis website metricset.
type Config struct {
	Names []string `config:"application_pool.name"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The iis application_pool metricset is beta.")
	var config Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	// instantiate reader object
	reader, err := newReader()
	if err != nil {
		return nil, err
	}

	if err := reader.initCounters(config.Names); err != nil {
		return nil, err
	}
	return &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger("application pool"),
		reader:        reader,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	var config Config
	if err := m.Module().UnpackConfig(&config); err != nil {
		return nil
	}

	events, err := m.reader.fetch(config.Names)
	if err != nil {
		return errors.Wrap(err, "failed reading counters")
	}

	for _, event := range events {
		isOpen := report.Event(event)
		if !isOpen {
			break
		}
	}
	return nil
}

// Close will be called when metricbeat is stopped, should close the query.
func (m *MetricSet) Close() error {
	err := m.reader.close()
	if err != nil {
		return errors.Wrap(err, "failed to close pdh query")
	}
	return nil
}
