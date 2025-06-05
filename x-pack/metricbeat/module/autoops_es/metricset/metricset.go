// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metricset

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

const MODULE_NAME = "autoops_es"

// This method should be invoked via any given MetricSet's init function to automatically register
// each AutoOpsMetricSet.
func AddAutoOpsMetricSet[T any](name string, routePath string, mapper EventsMapper[T]) {
	mb.Registry.MustAddMetricSet(MODULE_NAME, name, func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		return newAutoOpsMetricSet(base, routePath, mapper, nil)
	},
		mb.WithHostParser(elasticsearch.HostParser),
		mb.DefaultMetricSet(),
	)
}

// This method should be invoked via any given MetricSet's init function to automatically register
// each AutoOpsMetricSet that gets the elasticsearch.MetricSet passed to it.
func AddNestedAutoOpsMetricSet[T any](name string, routePath string, nestedMapper NestedEventsMapper[T]) {
	mb.Registry.MustAddMetricSet(MODULE_NAME, name, func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		return newAutoOpsMetricSet(base, routePath, nil, nestedMapper)
	},
		mb.WithHostParser(elasticsearch.HostParser),
		mb.DefaultMetricSet(),
	)
}

// Handle mapping the requested data and converting it into events.
type EventsMapper[T any] func(r mb.ReporterV2, info *utils.ClusterInfo, data *T) error

// Handle mapping the requested data and converting it into events.
type NestedEventsMapper[T any] func(m *elasticsearch.MetricSet, r mb.ReporterV2, info *utils.ClusterInfo, data *T) error

// AutoOpsMetricSet type defines all fields of the MetricSet
type AutoOpsMetricSet[T any] struct {
	*elasticsearch.MetricSet
	Mapper       EventsMapper[T]
	NestedMapper NestedEventsMapper[T]
	RoutePath    string
}

// New create a new instance of the AutoOpsMetricSet
func newAutoOpsMetricSet[T any](base mb.BaseMetricSet, routePath string, mapper EventsMapper[T], nestedMapper NestedEventsMapper[T]) (mb.MetricSet, error) {
	ms, err := elasticsearch.NewMetricSet(base, routePath)

	if err != nil {
		return nil, err
	}

	return &AutoOpsMetricSet[T]{
		Mapper:       mapper,
		MetricSet:    ms,
		NestedMapper: nestedMapper,
		RoutePath:    routePath,
	}, nil
}

// Fetch gathers stats for node using the _tasks API
func (m *AutoOpsMetricSet[T]) Fetch(r mb.ReporterV2) error {
	metricSetName := m.Name()

	m.Logger().Infof("fetching %v metricset", metricSetName)

	var err error
	var info *utils.ClusterInfo
	var data *T

	if info, err = GetInfo(m.MetricSet); err != nil {
		err = fmt.Errorf("failed to get cluster info from cluster, %v metricset %w", metricSetName, err)
		events.SendErrorEventWithoutClusterInfo(err, r, metricSetName)
		m.Logger().Errorf(err.Error())
		return err
	} else if data, err = utils.FetchAPIData[T](m.MetricSet, m.RoutePath); err != nil {
		err = fmt.Errorf("failed to get data, %v metricset %w", metricSetName, err)
		events.SendErrorEventWithoutClusterInfo(err, r, metricSetName)
		m.Logger().Errorf(err.Error())
		return err
	}

	// nested mappers reuse the
	if m.NestedMapper != nil {
		if err = m.NestedMapper(m.MetricSet, r, info, data); err != nil {
			return err
		}
	} else if err = m.Mapper(r, info, data); err != nil {
		return err
	}

	m.Logger().Infof("completed fetching %v metricset", metricSetName)
	return nil
}

// ensures that the type implements the interface
var _ mb.ReportingMetricSetV2Error = (*AutoOpsMetricSet[map[string]interface{}])(nil)
