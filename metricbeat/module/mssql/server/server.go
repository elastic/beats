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

// +build integration

package server

import (
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mssql"
	"github.com/pkg/errors"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("mssql", "server", New, mb.DefaultMetricSet())
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*mssql.MetricSet
	config *mssql.Config
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The mssql server metricset is experimental.")

	config := mssql.Config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	metricSet, err := mssql.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating mssql metricset")
	}

	ms := &MetricSet{MetricSet: metricSet, config: &config}

	return ms, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) {
	fetcher, err := mssql.NewFetcher(m.config, []string{
		"SELECT * FROM sys.dm_server_services;",
		"SELECT * FROM sys.dm_server_memory_dumps;",
	}, &schema)
	if err != nil {
		reporter.Error(errors.Wrap(err, "error creating fetcher"))
		return
	}
	defer fetcher.Close()

	if fetcher.Error != nil {
		reporter.Error(fetcher.Error)
	} else {
		for _, e := range fetcher.Maprs {
			reporter.Event(mb.Event{
				MetricSetFields: e,
			})
		}
	}
}
