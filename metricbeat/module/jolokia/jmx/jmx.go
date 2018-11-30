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
	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

var (
	metricsetName = "jolokia.jmx"
)

// init registers the MetricSet with the central registry.
func init() {
	if err := mb.Registry.AddMetricSet("jolokia", "jmx", New, hostParser); err != nil {
		panic(err)
	}
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
	http      JolokiaHTTPRequestFetcher
	log       *logp.Logger
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

	jolokiaHTTPBuild := NewJolokiaHTTPRequestFetcher(config.HTTPMethod)

	log := logp.NewLogger(metricsetName).With("host", base.HostData().Host)

	return &MetricSet{
		BaseMetricSet: base,
		mapping:       config.Mappings,
		namespace:     config.Namespace,
		http:          jolokiaHTTPBuild,
		log:           log,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	var allEvents []common.MapStr

	allEvents, err := m.http.Fetch(m)
	if err != nil {
		return nil, err
	}

	// Set dynamic namespace.
	var errs multierror.Errors
	for _, event := range allEvents {
		_, err := event.Put(mb.NamespaceKey, m.namespace)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return allEvents, errs.Err()
}
