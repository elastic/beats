package activity

import (
	"database/sql"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/postgresql"

	// Register postgresql database/sql driver
	_ "github.com/lib/pq"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("postgresql", "activity", New,
		mb.WithHostParser(postgresql.ParseURL),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the Postgresql MetricSet
type MetricSet struct {
	mb.BaseMetricSet
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{BaseMetricSet: base}, nil
}

// Fetch implements the data gathering and data conversion to the right format.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	db, err := sql.Open("postgres", m.HostData().URI)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	results, err := postgresql.QueryStats(db, "SELECT * FROM pg_stat_activity")
	if err != nil {
		return nil, errors.Wrap(err, "QueryStats")
	}

	events := []common.MapStr{}
	for _, result := range results {
		data, _ := schema.Apply(result)
		events = append(events, data)
	}

	return events, nil
}
