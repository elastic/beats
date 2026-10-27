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

// RequestOptions customizes the request that a ServerlessAutoOpsMetricSet makes, which is a plain GET without a body by default.
type RequestOptions struct {
	// The HTTP method to use, such as POST.
	Method string
	// The JSON body to send.
	Body []byte
}

// This method should be invoked via any given MetricSet's init function to automatically register
// each ServerlessAutoOpsMetricSet. It is not a default metricset, so it only runs when it is configured by name.
func AddServerlessAutoOpsMetricSet[T any](name string, routePath string, options RequestOptions, mapper EventsMapper[T]) {
	mb.Registry.MustAddMetricSet(MODULE_NAME, name, func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		return newServerlessAutoOpsMetricSet(base, routePath, options, mapper)
	},
		mb.WithHostParser(elasticsearch.HostParser),
	)
}

// ServerlessAutoOpsMetricSet is a metricset for an API that only exists on Elasticsearch Serverless. On any other cluster it silently
// does nothing, so it is safe to configure anywhere.
//
// Such APIs may need a request that is not the plain GET that every AutoOpsMetricSet makes, such as one with a body that lists what to
// return, so the request can be customized with RequestOptions.
//
// It is separate from AutoOpsMetricSet so that what only this kind of metricset needs does not affect the others. It follows the same
// flow (get the cluster info, fetch the data, map it to events), and shares everything that does not depend on the request: the cluster
// info, the error events, the Cloud Connected registration, and the HTTP helpers.
type ServerlessAutoOpsMetricSet[T any] struct {
	*elasticsearch.MetricSet
	Mapper    EventsMapper[T]
	RoutePath string
	Options   RequestOptions
}

// newServerlessAutoOpsMetricSet creates a new instance of the ServerlessAutoOpsMetricSet.
func newServerlessAutoOpsMetricSet[T any](base mb.BaseMetricSet, routePath string, options RequestOptions, mapper EventsMapper[T]) (mb.MetricSet, error) {
	ms, err := elasticsearch.NewMetricSet(base, routePath)

	if err != nil {
		return nil, err
	}

	// metricsets are not started asynchronously, so there is no need for synchronization here
	if !checkedCloudConnectedMode {
		checkedCloudConnectedMode = true

		if err := maybeRegisterCloudConnectedCluster(ms, GetInfo); err != nil {
			return nil, fmt.Errorf("failed to register Cloud Connected cluster: %w", err)
		}
	}

	return &ServerlessAutoOpsMetricSet[T]{
		MetricSet: ms,
		Mapper:    mapper,
		RoutePath: routePath,
		Options:   options,
	}, nil
}

// Fetch gets the data (on a serverless cluster) with the customized request and maps it to events.
func (m *ServerlessAutoOpsMetricSet[T]) Fetch(r mb.ReporterV2) error {
	metricSetName := m.Name()

	m.Logger().Debugf("fetching %v metricset", metricSetName)

	// like AutoOpsMetricSet, the errors are sent as events rather than returned, so that the metricset's Error() method is not called
	info, err := GetInfo(m.MetricSet)

	if err != nil {
		err = fmt.Errorf("failed to get cluster info from cluster, %v metricset %w", metricSetName, err)

		if utils.GetBoolEnvParam(GetClusterInfoValue()(), true) {
			events.LogAndSendErrorEventWithoutClusterInfo(err, r, metricSetName)
		} else {
			m.Logger().Errorf("Error fetching data for metricset %s: %s", metricSetName, err)
		}

		return nil
	}

	// the API does not exist on any other kind of cluster, so there is nothing to collect (and nothing worth reporting)
	if !info.IsServerless() {
		return nil
	}

	data, err := utils.FetchAPIDataWithRequest[T](m.MetricSet, m.Options.Method, m.RoutePath, m.Options.Body)

	if err != nil {
		err = fmt.Errorf("failed to get data, %v metricset %w", metricSetName, err)
		events.LogAndSendErrorEventWithoutClusterInfo(err, r, metricSetName)
		return nil
	}

	if err = m.Mapper(r, info, data); err != nil {
		return nil //nolint: nilerr // The error is reported by the mapper
	}

	m.Logger().Debugf("completed fetching %v metricset", metricSetName)
	return nil
}

// ensures that the type implements the interface
var _ mb.ReportingMetricSetV2Error = (*ServerlessAutoOpsMetricSet[map[string]any])(nil)
