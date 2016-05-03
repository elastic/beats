/*
Package status fetches MySQL server status metrics.

For more information on the query it uses, see:
http://dev.mysql.com/doc/refman/5.7/en/show-status.html
*/
package status

/*
TODO @ruflin, 20160315
 * Complete fields read
 * Complete template
 * Complete dashboards
*/

import (
	"database/sql"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mysql"

	"github.com/pkg/errors"
)

var (
	debugf = logp.MakeDebug("mysql-status")
)

func init() {
	if err := mb.Registry.AddMetricSet("mysql", "status", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching MySQL server status.
type MetricSet struct {
	mb.BaseMetricSet
	hostToDSN   map[string]string
	connections map[string]*sql.DB
}

// New creates and returns a new MetricSet instance.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Unpack additional configuration options.
	config := struct {
		Username string `config:"username"`
		Password string `config:"password"`
	}{
		Username: "",
		Password: "",
	}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	hostToDSN := make(map[string]string, len(base.Module().Config().Hosts))
	for _, host := range base.Module().Config().Hosts {
		// TODO (akroh): Apply validation to the mysql DSN format.
		dsn := mysql.CreateDSN(host, config.Username, config.Password)
		hostToDSN[host] = dsn
	}

	return &MetricSet{
		BaseMetricSet: base,
		hostToDSN:     hostToDSN,
		connections:   map[string]*sql.DB{},
	}, nil
}

// Fetch fetches status messages from mysql hosts.
func (m *MetricSet) Fetch(host string) (event common.MapStr, err error) {
	// TODO (akroh): reading and writing to map are not concurrent-safe
	db, found := m.connections[host]
	if !found {
		dsn, found := m.hostToDSN[host]
		if !found {
			return nil, fmt.Errorf("DSN not found for host '%s'", host)
		}

		var err error
		db, err = mysql.Connect(dsn)
		if err != nil {
			return nil, errors.Wrap(err, "mysql-status connect to host")
		}
		m.connections[host] = db
	}

	status, err := m.loadStatus(db)
	if err != nil {
		return nil, err
	}

	return eventMapping(status), nil
}

// loadStatus loads all status entries from the given database into an array.
func (m *MetricSet) loadStatus(db *sql.DB) (map[string]string, error) {
	// Returns the global status, also for versions previous 5.0.2
	rows, err := db.Query("SHOW /*!50002 GLOBAL */ STATUS;")
	if err != nil {
		return nil, err
	}

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
