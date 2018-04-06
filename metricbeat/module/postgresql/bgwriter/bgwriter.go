package bgwriter

import (
	"database/sql"
	"fmt"

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
	mb.Registry.MustAddMetricSet("postgresql", "bgwriter", New,
		mb.WithHostParser(postgresql.ParseURL),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
}

// New create a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{BaseMetricSet: base}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
func (m *MetricSet) Fetch() (common.MapStr, error) {
	db, err := sql.Open("postgres", m.HostData().URI)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	results, err := postgresql.QueryStats(db, "SELECT * FROM pg_stat_bgwriter")
	if err != nil {
		return nil, errors.Wrap(err, "QueryStats")
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("No results from the pg_stat_bgwriter query")
	}

	data, _ := schema.Apply(results[0])
	return data, nil
}
