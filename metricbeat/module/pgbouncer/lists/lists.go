package lists

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/pgbouncer"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// init registers the MetricSet with the central registry.
func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(0)
	mb.Registry.MustAddMetricSet("pgbouncer", "lists", New,
		mb.WithHostParser(pgbouncer.ParseURL),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*pgbouncer.MetricSet
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := pgbouncer.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It publishes the event which is then forwarded to the output. In case of an error, an error is reported.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	ctx := context.Background()
	results, err := m.QueryStats(ctx, "SHOW LISTS;")
	if err != nil {
		return fmt.Errorf("error in QueryStats: %w", err)
	}

	for _, result := range results {
		var data mapstr.M

		data, _ = schema.Apply(result)

		reporter.Event(mb.Event{
			MetricSetFields: data,
		})
	}
	return nil
}
