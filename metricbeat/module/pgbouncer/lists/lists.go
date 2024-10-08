package lists

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/pgbouncer"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	mb.Registry.MustAddMetricSet("pgbouncer", "lists", New,
		mb.WithHostParser(pgbouncer.ParseURL),
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	*pgbouncer.MetricSet
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := pgbouncer.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	ctx := context.Background()
	results, err := m.QueryStats(ctx, "SHOW LISTS;")
	if err != nil {
		return fmt.Errorf("error in QueryStats: %w", err)
	}

	for _, result := range results {
		data, err := schema.Apply(result)
		if err != nil {
			return fmt.Errorf("error applying schema: %w", err)
		}

		event := mapstr.M(data)
		reporter.Event(mb.Event{
			MetricSetFields: event,
		})
	}

	return nil
}
