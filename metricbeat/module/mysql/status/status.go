// Fetch status information from mysql: http://dev.mysql.com/doc/refman/5.7/en/show-status.html
package status

import (
	"database/sql"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/mysql"
	"github.com/urso/ucfg"
)

func init() {
	helper.Registry.AddMetricSeter("mysql", "status", New)
}

// New creates new instance of MetricSeter
func New() helper.MetricSeter {
	return &MetricSeter{
		connections: map[string]*sql.DB{},
	}
}

// MetricSetter object
type MetricSeter struct {
	connections map[string]*sql.DB
}

// Setup any metric specific configuration
func (m *MetricSeter) Setup(cfg *ucfg.Config) error {
	return nil
}

// Fetches status messages from mysql hosts
func (m *MetricSeter) Fetch(ms *helper.MetricSet) (events []common.MapStr, err error) {

	// Load status for all hosts and add it to events
	for _, host := range ms.Config.Hosts {
		db, err := m.getConnection(host)
		if err != nil {
			logp.Err("MySQL conenction error: %s", err)
		}

		status, err := m.loadStatus(db)

		if err != nil {
			return nil, err
		}

		event := eventMapping(status)
		events = append(events, event)
	}

	return events, nil
}

// getConnection returns the connection object for the given dsn
// In case a connection already exists it is reused
func (m *MetricSeter) getConnection(dsn string) (*sql.DB, error) {

	if db, ok := m.connections[dsn]; ok {
		return db, nil
	}

	db, err := mysql.Connect(dsn)
	if err != nil {
		return nil, err
	}

	m.connections[dsn] = db

	return db, nil
}

// loadStatus loads all status entries from the given database into an array
func (m *MetricSeter) loadStatus(db *sql.DB) (map[string]string, error) {

	rows, err := db.Query("SHOW STATUS")
	if err != nil {
		return nil, err
	}

	mysqlStatus := map[string]string{}

	for rows.Next() {
		var name string
		var value string
		rows.Scan(&name, &value)

		mysqlStatus[name] = value
	}

	return mysqlStatus, nil
}
