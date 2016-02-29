// Fetch status information from mysql: http://dev.mysql.com/doc/refman/5.7/en/show-status.html
package status

import (
	"database/sql"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/mysql"
)

func init() {
	helper.Registry.AddMetricSeter("mysql", "status", MetricSeter{})
}

// MetricSetter object
type MetricSeter struct {
}

func (m MetricSeter) Setup() error {
	return nil
}

// Fetches status messages from mysql hosts
func (m MetricSeter) Fetch(ms *helper.MetricSet) (events []common.MapStr, err error) {

	// Load status for all hosts and add it to events
	for _, host := range ms.Config.Hosts {
		db, err := mysql.Connect(host)
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

func (m MetricSeter) Cleanup() error {
	return nil
}

// loadStatus loads all status entries from the given database into an array
func (m MetricSeter) loadStatus(db *sql.DB) (map[string]string, error) {

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
