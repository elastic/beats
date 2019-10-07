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

package dbstats

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mongodb"
)

var logger = logp.NewLogger("mongodb.dbstats")

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("mongodb", "dbstats", New,
		mb.WithHostParser(mongodb.ParseURL),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	*mongodb.MetricSet
}

// New creates a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := mongodb.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	// instantiate direct connections to each of the configured Mongo hosts
	mongoSession, err := mongodb.NewDirectSession(m.DialInfo)
	if err != nil {
		return errors.Wrap(err, "error creating new Session")
	}
	defer mongoSession.Close()

	// Get the list of databases names, which we'll use to call db.stats() on each
	dbNames, err := mongoSession.DatabaseNames()
	if err != nil {
		return errors.Wrap(err, "Error retrieving database names from Mongo instance")
	}

	// for each database, call db.stats() and append to events
	totalEvents := 0
	for _, dbName := range dbNames {
		db := mongoSession.DB(dbName)

		result := common.MapStr{}

		err := db.Run("dbStats", &result)
		if err != nil {
			err = errors.Wrapf(err, "Failed to retrieve stats for db %s", dbName)
			reporter.Error(err)
			m.Logger().Error(err)
			continue
		}
		data, _ := schema.Apply(result)
		reported := reporter.Event(mb.Event{MetricSetFields: data})
		if !reported {
			return nil
		}
		totalEvents++
	}

	if totalEvents == 0 {
		return errors.New("Failed to retrieve dbStats from any databases")

	}

	return nil
}
