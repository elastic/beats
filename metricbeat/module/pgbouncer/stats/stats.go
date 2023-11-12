package stats
import (
	"context"
	"fmt"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/postgresql"
)
// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
    mb.Registry.MustAddMetricSet("pgbouncer", "stats", New,
        mb.WithHostParser(postgresql.ParseURL),
        mb.DefaultMetricSet(),
    )
}
// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*postgresql.MetricSet
}
// New create a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := postgresql.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	ctx := context.Background()
	results, err := m.QueryStats(ctx, "SHOW STATS;")
	if err != nil {
		return fmt.Errorf("error in QueryStats: %w", err)
	}
	if len(results) == 0 {
		return fmt.Errorf("No results from the stats query")
	}
	data, _ := schema.Apply(results[0])
	reporter.Event(mb.Event{
		MetricSetFields: data,
	})
	return nil
}