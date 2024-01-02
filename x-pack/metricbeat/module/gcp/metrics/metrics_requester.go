// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
)

type metricsRequester struct {
	config config

	client *monitoring.MetricClient

	logger *logp.Logger
}

type timeSeriesWithAligner struct {
	timeSeries []*monitoringpb.TimeSeries
	aligner    string
}

func (r *metricsRequester) Metric(ctx context.Context, serviceName, metricType string, timeInterval *monitoringpb.TimeInterval, aligner string) timeSeriesWithAligner {
	timeSeries := make([]*monitoringpb.TimeSeries, 0)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:     "projects/" + r.config.ProjectID,
		Interval: timeInterval,
		View:     monitoringpb.ListTimeSeriesRequest_FULL,
		Filter:   r.getFilterForMetric(serviceName, metricType),
		Aggregation: &monitoringpb.Aggregation{
			PerSeriesAligner: gcp.AlignersMapToGCP[aligner],
			AlignmentPeriod:  r.config.period,
		},
	}

	it := r.client.ListTimeSeries(ctx, req)

	for {
		resp, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			r.logger.Errorf("Could not read time series value: %s: %v", metricType, err)
			break
		}

		timeSeries = append(timeSeries, resp)
	}

	out := timeSeriesWithAligner{
		aligner:    aligner,
		timeSeries: timeSeries,
	}

	return out
}

func (r *metricsRequester) Metrics(ctx context.Context, serviceName string, aligner string, metricsToCollect map[string]metricMeta) ([]timeSeriesWithAligner, error) {
	var lock sync.Mutex
	var wg sync.WaitGroup
	results := make([]timeSeriesWithAligner, 0)

	// Find the largest delay in the metrics to collect.
	//
	// Why do we need find the largest ingest delay in the metrics to collect?
	// ======================================================================
	//
	// We need to share some context first.
	//
	// Context
	// -------
	//
	// GCP metrics have different ingestion delays; some metrics have zero delay,
	// while others have a non-zero delay of up to a few minutes.
	//
	// For example,
	//  - `container/memory.limit.bytes` has no ingest delay.
	//  - `container/memory/request_bytes` has two minutes ingest delay.
	//
	// Since the metricset collects metrics every 60 seconds, it ends up
	// collecting `container/memory.limit.bytes` and `container/memory/request_bytes`
	// in different iterations; it stores metrics values in different documents,
	// even when they are related to the same timestamp.
	//
	// Problem
	// -------
	//
	// When TSDB is enabled, two documents cannot have the same timestamp and dimensions.
	// If they do, the second document is dropped.
	//
	// Unfortunately, this is exactly what happens when the metricset collects
	// `container/memory.limit.bytes` and `container/memory/request_bytes` in different
	// iterations.
	//
	// Solution
	// --------
	//
	// We calculate the largest delay, and then we collect the metrics values only when
	// they are all available.
	//
	largestDelay := 0 * time.Second
	for _, meta := range metricsToCollect {
		metricMeta := meta
		if meta.ingestDelay > largestDelay {
			largestDelay = metricMeta.ingestDelay
		}
	}

	for mt, meta := range metricsToCollect {
		wg.Add(1)

		metricMeta := meta
		go func(mt string) {
			defer wg.Done()

			r.logger.Debugf("For metricType %s, metricMeta = %d,  aligner = %s", mt, metricMeta, aligner)
			interval, aligner := getTimeIntervalAligner(largestDelay, metricMeta.samplePeriod, r.config.period, aligner)
			ts := r.Metric(ctx, serviceName, mt, interval, aligner)
			lock.Lock()
			defer lock.Unlock()
			results = append(results, ts)
		}(mt)
	}

	wg.Wait()
	return results, nil
}

func (r *metricsRequester) buildRegionsFilter(regions []string, label string) string {
	if len(regions) == 0 {
		return ""
	}

	var filter strings.Builder

	// No. of regions added to the filter string.
	var regionsCount uint

	for _, region := range regions {
		// If 1 region has been added and the iteration continues, add the OR operator.
		if regionsCount > 0 {
			filter.WriteString("OR")
			filter.WriteString(" ")
		}

		filter.WriteString(fmt.Sprintf("%s = starts_with(\"%s\")", label, trimWildcard(region)))
		filter.WriteString(" ")

		regionsCount++
	}

	switch {
	// If the filter string has more than 1 region, parentheses are added for better filter readability.
	case regionsCount > 1:
		return fmt.Sprintf("(%s)", strings.TrimSpace(filter.String()))
	default:
		return strings.TrimSpace(filter.String())
	}
}

// getServiceLabelFor return the appropriate label to use for filtering metrics of a given service.
func getServiceLabelFor(serviceName string) string {
	switch serviceName {
	case gcp.ServiceCompute:
		return gcp.ComputeResourceLabel
	case gcp.ServiceGKE:
		return gcp.GKEResourceLabel
	case gcp.ServiceDataproc:
		return gcp.DataprocResourceLabel
	case gcp.ServiceStorage:
		return gcp.StorageResourceLabel
	case gcp.ServiceCloudSQL:
		return gcp.CloudSQLResourceLabel
	case gcp.ServiceRedis:
		return gcp.RedisResourceLabel
	default:
		return gcp.DefaultResourceLabel
	}
}

func (r *metricsRequester) buildLocationFilter(serviceLabel, filter string) string {
	if r.config.Region != "" && r.config.Zone != "" {
		r.logger.Warnf("when region %s and zone %s config parameter "+
			"both are provided, only use region", r.config.Regions, r.config.Zone)
	}

	if r.config.Region != "" && len(r.config.Regions) != 0 {
		r.logger.Warnf("when region %s and regions config parameters are both provided, use region", r.config.Region)
	}

	switch {
	case r.config.Region != "":
		filter = fmt.Sprintf("%s AND %s = starts_with(\"%s\")", filter, serviceLabel, trimWildcard(r.config.Region))
	case r.config.Zone != "":
		filter = fmt.Sprintf("%s AND %s = starts_with(\"%s\")", filter, serviceLabel, trimWildcard(r.config.Zone))
	case len(r.config.Regions) != 0:
		regionsFilter := r.buildRegionsFilter(r.config.Regions, serviceLabel)
		filter = fmt.Sprintf("%s AND %s", filter, regionsFilter)
	}

	return filter
}

// trimWildcard remove wildcard value `*` from the end of the string.
func trimWildcard(value string) string {
	return strings.TrimSuffix(value, "*")
}

// isAGlobalService return true if the given service is considered global from GCP and does not
// uses the regional or zonal metrics filtering.
func isAGlobalService(serviceName string) bool {
	switch serviceName {
	case gcp.ServicePubsub, gcp.ServiceLoadBalancing, gcp.ServiceCloudFunctions, gcp.ServiceFirestore:
		return true
	default:
		return false
	}
}

// getFilterForMetric returns the filter associated with the corresponding filter. Some services like Pub/Sub fails
// if they have a region specified.
func (r *metricsRequester) getFilterForMetric(serviceName, m string) string {
	f := fmt.Sprintf(`metric.type="%s"`, m)

	locationsConfigsAvailable := r.config.Region != "" || r.config.Zone != "" || len(r.config.Regions) > 0
	// NOTE: some GCP services are global, not regional or zonal. To these services we don't need
	// to apply any additional filters.
	if locationsConfigsAvailable && !isAGlobalService(serviceName) {
		serviceLabel := getServiceLabelFor(serviceName)
		f = r.buildLocationFilter(serviceLabel, f)
	}

	// NOTE: make sure to log the applied filter, as it helpful when debugging
	r.logger.Debugf("ListTimeSeries API filter = %s", f)

	return f
}

// Returns a GCP TimeInterval based on the ingestDelay and samplePeriod from ListMetricDescriptor
func getTimeIntervalAligner(ingestDelay time.Duration, samplePeriod time.Duration, collectionPeriod *durationpb.Duration, inputAligner string) (*monitoringpb.TimeInterval, string) {
	var startTime, endTime, currentTime time.Time
	var needsAggregation bool
	currentTime = time.Now().UTC()

	// When samplePeriod < collectionPeriod, aggregation will be done in ListTimeSeriesRequest.
	// For example, samplePeriod = 60s, collectionPeriod = 300s, if perSeriesAligner is not given,
	// ALIGN_MEAN will be used by default.
	if int64(samplePeriod.Seconds()) < collectionPeriod.Seconds {
		endTime = currentTime.Add(-ingestDelay)
		startTime = endTime.Add(-time.Duration(collectionPeriod.Seconds) * time.Second)
		needsAggregation = true
	}

	// When samplePeriod == collectionPeriod, aggregation is not needed
	// When samplePeriod > collectionPeriod, aggregation is not needed, use sample period
	// to determine startTime and endTime to make sure there will be data point in this time range.
	if int64(samplePeriod.Seconds()) >= collectionPeriod.Seconds {
		endTime = currentTime.Add(-ingestDelay)
		startTime = endTime.Add(-samplePeriod)
		needsAggregation = false
	}

	interval := &monitoringpb.TimeInterval{
		StartTime: &timestamppb.Timestamp{
			Seconds: startTime.Unix(),
		},
		EndTime: &timestamppb.Timestamp{
			Seconds: endTime.Unix(),
		},
	}

	// Default aligner for aggregation is ALIGN_NONE if it's not given
	updatedAligner := gcp.DefaultAligner
	if needsAggregation && inputAligner != "" {
		updatedAligner = inputAligner
	}

	return interval, updatedAligner
}
