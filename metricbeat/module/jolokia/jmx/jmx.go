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

package jmx

import (
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

var (
	metricsetName = "jolokia.jmx"
)

// init registers the MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("jolokia", "jmx", New,
		mb.WithHostParser(hostParser),
	)
}

const (
	defaultScheme = "http"
	defaultPath   = "/jolokia/"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: "path",
		DefaultPath:   defaultPath,
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	mapping   []JMXMapping
	namespace string
	jolokia   JolokiaHTTPRequestFetcher
	log       *logp.Logger
	http      *helper.HTTP
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct {
		Namespace  string       `config:"namespace" validate:"required"`
		HTTPMethod string       `config:"http_method"`
		Mappings   []JMXMapping `config:"jmx.mappings" validate:"required"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	jolokiaFetcher := NewJolokiaHTTPRequestFetcher(config.HTTPMethod)

	log := logp.NewLogger(metricsetName).With("host", base.HostData().Host)

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		mapping:       config.Mappings,
		namespace:     config.Namespace,
		jolokia:       jolokiaFetcher,
		log:           log,
		http:          http,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	var allEvents []mapstr.M

	allEvents, err := m.jolokia.Fetch(m)
	if err != nil {
		return err
	}

	// Set dynamic namespace.
	for _, event := range allEvents {
		reporter.Event(mb.Event{
			MetricSetFields: event,
			Namespace:       m.Module().Name() + "." + m.namespace,
		})
	}
	return nil
}
