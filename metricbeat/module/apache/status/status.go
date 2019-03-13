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

// Package status reads Apache HTTPD server status from the mod_status module.
package status

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	// defaultScheme is the default scheme to use when it is not specified in
	// the host config.
	defaultScheme = "http"

	// defaultPath is the default path to the mod_status endpoint on the
	// Apache HTTPD server.
	defaultPath = "/server-status"

	// autoQueryParam is a query parameter added to the request so that
	// mod_status returns machine-readable output.
	autoQueryParam = "auto"
)

var (
	debugf = logp.MakeDebug("apache-status")

	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: "server_status_path",
		DefaultPath:   defaultPath,
		QueryParams:   autoQueryParam,
	}.Build()
)

func init() {
	mb.Registry.MustAddMetricSet("apache", "status", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching Apache HTTPD server status.
type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP
}

// New creates new instance of MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		base,
		http,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	scanner, err := m.http.FetchScanner()
	if err != nil {
		return errors.Wrap(err, "error fetching data")
	}

	data, _ := eventMapping(scanner, m.Host())

	if reported := reporter.Event(mb.Event{MetricSetFields: data}); !reported {
		m.Logger().Error("error reporting event")
	}

	return nil
}
