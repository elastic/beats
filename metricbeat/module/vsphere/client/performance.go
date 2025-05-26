// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/elastic/elastic-agent-libs/logp"
)

type PerfManager interface {
	AvailableMetric(ctx context.Context, entity types.ManagedObjectReference, interval int32) (performance.MetricList, error)
	Query(ctx context.Context, spec []types.PerfQuerySpec) ([]types.BasePerfEntityMetricBase, error)
	ToMetricSeries(ctx context.Context, series []types.BasePerfEntityMetricBase) ([]performance.EntityMetric, error)
}

type PerformanceDataFetcher struct {
	perfManager PerfManager
	logger      *logp.Logger
}

func NewPerformanceDataFetcher(logger *logp.Logger, perfManager PerfManager) *PerformanceDataFetcher {
	return &PerformanceDataFetcher{
		logger:      logger,
		perfManager: perfManager,
	}
}

func (p *PerformanceDataFetcher) GetPerfMetrics(ctx context.Context,
	period int32,
	objectType string,
	objectName string,
	objectReference types.ManagedObjectReference,
	metrics map[string]*types.PerfCounterInfo,
	metricSet map[string]struct{}) (metricMap map[string]interface{}, err error) {

	metricMap = make(map[string]interface{})

	availableMetric, err := p.perfManager.AvailableMetric(ctx, objectReference, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get available metrics: %w", err)
	}

	availableMetricByKey := availableMetric.ByKey()

	// Filter for required metrics
	var metricIDs []types.PerfMetricId
	for key, metric := range metricSet {
		if counter, ok := metrics[key]; ok {
			if _, exists := availableMetricByKey[counter.Key]; exists {
				metricIDs = append(metricIDs, types.PerfMetricId{
					CounterId: counter.Key,
					Instance:  "*",
				})
			}
		} else {
			p.logger.Warnf("Metric %s not found", metric)
		}
	}

	spec := types.PerfQuerySpec{
		Entity:     objectReference,
		MetricId:   metricIDs,
		MaxSample:  1,
		IntervalId: period,
	}

	// Query performance data
	samples, err := p.perfManager.Query(ctx, []types.PerfQuerySpec{spec})
	if err != nil {
		if strings.Contains(err.Error(), "ServerFaultCode: A specified parameter was not correct: querySpec.interval") {
			return metricMap, fmt.Errorf("failed to query performance data: use one of the system's supported interval. consider adjusting period: %w", err)
		}

		return metricMap, fmt.Errorf("failed to query performance data: %w", err)
	}

	if len(samples) == 0 {
		p.logger.Debug("No samples returned from performance manager")
		return metricMap, nil
	}

	results, err := p.perfManager.ToMetricSeries(ctx, samples)
	if err != nil {
		return metricMap, fmt.Errorf("failed to convert performance data to metric series: %w", err)
	}

	if len(results) == 0 {
		p.logger.Debug("No results returned from metric series conversion")
		return metricMap, nil
	}

	for _, result := range results[0].Value {
		if len(result.Value) > 0 {
			value := result.Value[0]
			if result.Unit == string(types.PerformanceManagerUnitPercent) {
				metricMap[result.Name] = float64(value) / 100.0
			} else {
				metricMap[result.Name] = value
			}
			continue
		}
		p.logger.Debugf("For %s %s, Metric %s: No result found", objectType, objectName, result.Name)
	}

	return metricMap, nil
}
