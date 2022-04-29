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

// Package stubstatus reads server status from nginx host under /server-status, ngx_http_stub_status_module is required.
package stubstatus

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	// defaultScheme is the default scheme to use when it is not specified in
	// the host config.
	defaultScheme = "http"

	// defaultPath is the default path to the ngx_http_stub_status_module endpoint on Nginx.
	defaultPath = "/nginx_status"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: "server_status_path",
		DefaultPath:   defaultPath,
	}.Build()
)

var logger = logp.NewLogger("nginx.stubstatus")

func init() {
	mb.Registry.MustAddMetricSet("nginx", "stubstatus", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching Nginx stub status.
type MetricSet struct {
	mb.BaseMetricSet
	http                *helper.HTTP
	previousNumRequests int // Total number of requests as returned in the previous fetch.
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		BaseMetricSet: base,
		http:          http,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	scanner, err := m.http.FetchScanner()
	if err != nil {
		return errors.Wrap(err, "error fetching status")
	}
	event, _ := eventMapping(scanner, m)
	reporter.Event(mb.Event{MetricSetFields: event})

	return nil
}
