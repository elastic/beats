// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type observedService struct {
	next    Service
	metrics *azureAPIMetrics
}

func newObservedService(next Service, metrics *azureAPIMetrics) Service {
	if metrics == nil {
		return next
	}
	return &observedService{
		next:    next,
		metrics: metrics,
	}
}

func newObservedAzureMonitorService(next Service, registry *monitoring.Registry, logger *logp.Logger) Service {
	return newObservedService(next, newAzureAPIMetrics(registry, logger))
}

func (s *observedService) GetResourceDefinitionById(id string) (resp armresources.GenericResource, err error) {
	start := time.Now()
	defer func() {
		s.metrics.observe(operationGetResourceDefinitionByID, time.Since(start).Nanoseconds(), err)
	}()

	return s.next.GetResourceDefinitionById(id)
}

func (s *observedService) GetResourceDefinitions(id []string, group []string, rType string, query string) (resp []*armresources.GenericResourceExpanded, err error) {
	start := time.Now()
	defer func() {
		s.metrics.observe(operationGetResourceDefinitions, time.Since(start).Nanoseconds(), err)
	}()

	return s.next.GetResourceDefinitions(id, group, rType, query)
}

func (s *observedService) GetMetricDefinitionsWithRetry(resourceId string, namespace string) (resp armmonitor.MetricDefinitionCollection, err error) {
	start := time.Now()
	defer func() {
		s.metrics.observe(operationGetMetricDefinitions, time.Since(start).Nanoseconds(), err)
	}()

	return s.next.GetMetricDefinitionsWithRetry(resourceId, namespace)
}

func (s *observedService) GetMetricNamespaces(resourceId string) (resp armmonitor.MetricNamespaceCollection, err error) {
	start := time.Now()
	defer func() {
		s.metrics.observe(operationGetMetricNamespaces, time.Since(start).Nanoseconds(), err)
	}()

	return s.next.GetMetricNamespaces(resourceId)
}

func (s *observedService) GetMetricValues(
	resourceId string,
	namespace string,
	timegrain string,
	timespan string,
	metricNames []string,
	aggregations string,
	filter string,
) (resp []armmonitor.Metric, responseTimegrain string, err error) {
	start := time.Now()
	defer func() {
		s.metrics.observe(operationGetMetricValues, time.Since(start).Nanoseconds(), err)
	}()

	return s.next.GetMetricValues(resourceId, namespace, timegrain, timespan, metricNames, aggregations, filter)
}

func (s *observedService) QueryResources(
	resourceIDs []string,
	subscriptionID string,
	namespace string,
	timegrain string,
	startTime string,
	endTime string,
	metricNames []string,
	aggregations string,
	filter string,
	location string,
) (resp []azmetrics.MetricData, err error) {
	start := time.Now()
	defer func() {
		s.metrics.observe(operationQueryResources, time.Since(start).Nanoseconds(), err)
	}()

	return s.next.QueryResources(resourceIDs, subscriptionID, namespace, timegrain, startTime, endTime, metricNames, aggregations, filter, location)
}

var _ Service = (*observedService)(nil)
