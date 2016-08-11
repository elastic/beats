package activity

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("postgresql", "activity", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the Postgresql MetricSet
type MetricSet struct {
	mb.BaseMetricSet
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct{}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch implements the data gathering and data conversion to the right format.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	// TODO: Find a way to pass the timeout
	db, err := sql.Open("postgres", m.Host())
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM pg_stat_activity")
	if err != nil {
		return nil, err
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.Wrap(err, "scanning columns")
	}
	vals := make([][]byte, len(columns))
	valPointers := make([]interface{}, len(columns))
	for i, _ := range vals {
		valPointers[i] = &vals[i]
	}

	events := []common.MapStr{}

	for rows.Next() {
		err = rows.Scan(valPointers...)
		if err != nil {
			return nil, errors.Wrap(err, "scanning row")
		}

		result := map[string]interface{}{}
		for i, col := range columns {
			result[col] = string(vals[i])
		}

		logp.Debug("postgresql", "Result: %v", result)
		events = append(events, eventMapping(result))
	}

	return events, nil
}
