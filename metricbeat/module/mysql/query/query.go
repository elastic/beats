package query

import (
	"database/sql"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mysql"

	"github.com/pkg/errors"
)

var (
	debugf = logp.MakeDebug("mysql-query")
)

func init() {
	if err := mb.Registry.AddMetricSet("mysql", "query", New, mysql.ParseDSN); err != nil {
		panic(err)
	}
}

// MetricSet for fetching MySQL server query.
type MetricSet struct {
	mb.BaseMetricSet
	db *sql.DB
	query string
	namespace string
}

// New creates and returns a new MetricSet instance.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	// Unpack additional configuration options.
	config := struct {
		Query string `config:"query"`
		Namespace string `config:"namespace"`
	}{}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		query: config.Query,
		namespace: config.Namespace,
	}, nil
}

// Fetch fetches query messages from a mysql host.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	if m.db == nil {
		var err error
		m.db, err = mysql.NewDB(m.HostData().URI)
		if err != nil {
			return nil, errors.Wrap(err, "mysql-query fetch failed")
		}
	}

	event, err := m.loadQuery(m.db, m.query)
	event["_metricsetName"] = m.namespace

	if err != nil {
		return event, err
	}

	return event, nil
}

// loadStatus loads all status entries from the given database into an array.
func (m *MetricSet) loadQuery(db *sql.DB, query string) (common.MapStr, error) {
	// Returns all rows for the given query
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := common.MapStr{}

	for rows.Next() {
		var name string
		var value string

		err = rows.Scan(&name, &value)
		if err != nil {
			return nil, err
		}

		data[name] = value
	}

	return data, nil
}
