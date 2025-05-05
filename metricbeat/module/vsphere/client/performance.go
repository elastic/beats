package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/vim25/types"
)

type PerformanceDataFetcher struct {
	perfManager *performance.Manager
	logger      *logp.Logger
}

func NewPerformanceDataFetcher(logger *logp.Logger, perfManager *performance.Manager) *PerformanceDataFetcher {
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
			if objectType == "virtualMachine" {
				p.logger.Infof("METRIC RESULT: %+v", result)
			}
			metricMap[result.Name] = result.Value[0]
			continue
		}
		p.logger.Debugf("For %s %s, Metric %s: No result found", objectType, objectName, result.Name)
	}

	return metricMap, nil
}
