/*
Package status fetches MySQL server status metrics.

For more information on the query it uses, see:
http://dev.mysql.com/doc/refman/5.7/en/show-status.html
*/
package status

import (
	"database/sql"

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
	dsn string
	db  *sql.DB
}

// New creates and returns a new MetricSet instance.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Unpack additional configuration options.
	config := struct {
		Hosts    []string `config:"hosts"    validate:"nonzero,required"`
		Username string   `config:"username"`
		Password string   `config:"password"`
	}{
		Username: "",
		Password: "",
	}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	// Create and validate the data source name.
	dsn, err := mysql.CreateDSN(base.Host(), config.Username, config.Password, base.Module().Config().Timeout)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		dsn:           dsn,
	}, nil
}

// Fetch fetches status messages from a mysql host.
func (m *MetricSet) Fetch() (event common.MapStr, err error) {
	if m.db == nil {
		var err error
		m.db, err = mysql.NewDB(m.dsn)
		if err != nil {
			return nil, errors.Wrap(err, "mysql-status fetch failed")
		}
	}

	status, err := m.loadStatus(m.db)
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
