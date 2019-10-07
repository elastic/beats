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

/*
Package galera_status fetches MySQL Galera server status metrics.

For more information on the query it uses, see:
http://dev.mysql.com/doc/refman/5.7/en/show-status.html
http://galeracluster.com/documentation-webpages/galerastatusvariables.html
*/
package galera_status

import (
	"database/sql"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mysql"

	"github.com/pkg/errors"
)

// init registers the MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("mysql", "galera_status", New,
		mb.WithHostParser(mysql.ParseDSN),
	)
}

// MetricSet for fetching Galera-MySQL server status
type MetricSet struct {
	mb.BaseMetricSet
	db *sql.DB
}

// New create a new instance of the MetricSet
// Loads query_mode config setting from the config file
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The galera_status metricset is experimental.")
	return &MetricSet{BaseMetricSet: base}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	if m.db == nil {
		var err error
		m.db, err = mysql.NewDB(m.HostData().URI)
		if err != nil {
			return errors.Wrap(err, "Galera-status fetch failed")
		}
	}

	status, err := m.loadStatus(m.db)
	if err != nil {
		return err
	}

	event := eventMapping(status)

	if m.Module().Config().Raw {
		event["raw"] = rawEventMapping(status)
	}

	reporter.Event(mb.Event{
		MetricSetFields: event,
	})

	return nil
}

// loadStatus loads all status entries from the given database into an array.
func (m *MetricSet) loadStatus(db *sql.DB) (map[string]string, error) {
	// Returns the global status, also for versions previous 5.0.2
	rows, err := db.Query("SHOW /*!50002 GLOBAL */ STATUS LIKE 'wsrep_%';")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	galeraStatus := map[string]string{}

	for rows.Next() {
		var name string
		var value string

		err = rows.Scan(&name, &value)
		if err != nil {
			return nil, err
		}

		galeraStatus[name] = value
	}

	return galeraStatus, nil
}

// Close closes the database connection and prevents future queries.
func (m *MetricSet) Close() error {
	if m.db == nil {
		return nil
	}
	return errors.Wrap(m.db.Close(), "failed to close mysql database client")
}
