package statement

import (
	"database/sql"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/postgresql"

	// Register postgresql database/sql driver
	_ "github.com/lib/pq"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("postgresql", "statement", New,
		mb.WithHostParser(postgresql.ParseURL),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The statement metricset is beta")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	db, err := sql.Open("postgres", m.HostData().URI)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	results, err := postgresql.QueryStats(db, "SELECT * FROM pg_stat_statements")
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
