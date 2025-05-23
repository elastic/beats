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
Package status fetches MySQL server status metrics.

For more information on the query it uses, see:
http://dev.mysql.com/doc/refman/5.7/en/show-status.html
*/
package status

import (
	"database/sql"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/mysql"
)

func init() {
	mb.Registry.MustAddMetricSet("mysql", "status", New,
		mb.WithHostParser(mysql.ParseDSN),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching MySQL server status.
type MetricSet struct {
	*mysql.Metricset
	db *sql.DB
}

// New creates and returns a new MetricSet instance.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := mysql.NewMetricset(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{Metricset: ms, db: nil}, nil
}

// Fetch fetches status messages from a mysql host.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	if m.db == nil {
		var err error
		m.db, err = mysql.NewDB(m.HostData().URI, m.Metricset.Config.TLSConfig)
		if err != nil {
			return fmt.Errorf("mysql-status fetch failed: %w", err)
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
	rows, err := db.Query("SHOW /*!50002 GLOBAL */ STATUS;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mysqlStatus := map[string]string{}

	for rows.Next() {
		var name string
		var value string

		err = rows.Scan(&name, &value)
		if err != nil {
			return nil, err
		}

		mysqlStatus[name] = value
	}

	return mysqlStatus, nil
}

// Close closes the database connection and prevents future queries.
func (m *MetricSet) Close() error {
	if m.db == nil {
		return nil
	}
	return fmt.Errorf("failed to close mysql database client: %w", m.db.Close())
}
