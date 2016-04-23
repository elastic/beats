/**

Fetch status information from mysql: http://dev.mysql.com/doc/refman/5.7/en/show-status.html

TODO @ruflin, 20160315
 * Complete fields read
 * Complete template
 * Complete dashboards

*/
package status

import (
	"database/sql"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/mysql"
)

func init() {
	if err := helper.Registry.AddMetricSeter("mysql", "status", New); err != nil {
		panic(err)
	}
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
func (m *MetricSeter) Setup(ms *helper.MetricSet) error {

	// Additional configuration options
	config := struct {
		Username string `config:"username"`
		Password string `config:"password"`
	}{
		Username: "",
		Password: "",
	}

	for _, host := range ms.Config.Hosts {
		dsn := mysql.CreateDSN(host, config.Username, config.Password)

		db, err := mysql.Connect(dsn)
		if err != nil {
			logp.Err(err.Error())
		}

		m.connections[host] = db
	}

	return nil
}

// Fetches status messages from mysql hosts
func (m *MetricSeter) Fetch(ms *helper.MetricSet, host string) (event common.MapStr, err error) {

	db := m.connections[host]

	status, err := m.loadStatus(db)

	if err != nil {
		return nil, err
	}

	return eventMapping(status), nil

}

// loadStatus loads all status entries from the given database into an array
func (m *MetricSeter) loadStatus(db *sql.DB) (map[string]string, error) {

	// Returns the global status, also for versions previous 5.0.2
	rows, err := db.Query("SHOW /*!50002 GLOBAL */ STATUS;")
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
