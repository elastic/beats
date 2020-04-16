// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stackdriver

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/duration"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/googlecloud"
)

type stackdriverMetricsRequester struct {
	config config

	client *monitoring.MetricClient

	logger *logp.Logger
}

func (r *stackdriverMetricsRequester) Metric(ctx context.Context, m string, timeInterval *monitoringpb.TimeInterval, needsAggregation bool) (out []*monitoringpb.TimeSeries) {
	out = make([]*monitoringpb.TimeSeries, 0)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:        "projects/" + r.config.ProjectID,
		Interval:    timeInterval,
		View:        monitoringpb.ListTimeSeriesRequest_FULL,
		Filter:      r.getFilterForMetric(m),
		Aggregation: constructAggregation(r.config.period, r.config.PerSeriesAligner, needsAggregation),
	}

	it := r.client.ListTimeSeries(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			r.logger.Errorf("Could not read time series value: %s: %v", m, err)
			break
		}

		out = append(out, resp)
	}

	return
}

func constructFilter(m string, region string, zone string) string {
	filter := fmt.Sprintf(`metric.type="%s" AND resource.labels.zone = `, m)
	// If region is specified, use region as filter resource label.
	// If region is empty but zone is given, use zone instead.
	if region != "" {
		filter += fmt.Sprintf(`starts_with("%s")`, region)
	} else if zone != "" {
		filter += fmt.Sprintf(`"%s"`, zone)
	}
	return filter
}

func (r *stackdriverMetricsRequester) Metrics(ctx context.Context, metricTypes []string, metricsMeta map[string]metricMeta) ([]*monitoringpb.TimeSeries, error) {
	var lock sync.Mutex
	var wg sync.WaitGroup
	results := make([]*monitoringpb.TimeSeries, 0)

	for _, mt := range metricTypes {
		metricType := mt
		wg.Add(1)

		go func(metricType string) {
			defer wg.Done()

			metricMeta := metricsMeta[metricType]
			interval, needsAggregation := getTimeInterval(metricMeta.ingestDelay, metricMeta.samplePeriod, r.config.period)
			ts := r.Metric(ctx, metricType, interval, needsAggregation)

			lock.Lock()
			defer lock.Unlock()
			results = append(results, ts...)
		}(metricType)
	}

	wg.Wait()
	return results, nil
}

var serviceRegexp = regexp.MustCompile(`^(?P<service>[a-z]+)\.googleapis.com.*`)

// getFilterForMetric returns the filter associated with the corresponding filter. Some services like Pub/Sub fails
// if they have a region specified.
func (r *stackdriverMetricsRequester) getFilterForMetric(m string) (f string) {
	f = fmt.Sprintf(`metric.type="%s"`, m)

	service := serviceRegexp.ReplaceAllString(m, "${service}")

	switch service {
	case googlecloud.ServicePubsub, googlecloud.ServiceLoadBalancing:
		return
	case googlecloud.ServiceStorage:
		if r.config.Region == "" {
			return
		}

		f = fmt.Sprintf(`%s AND resource.labels.location = "%s"`, f, r.config.Region)
	default:
		if r.config.Region != "" && r.config.Zone != "" {
			r.logger.Warnf("when region %s and zone %s config parameter "+
				"both are provided, only use region", r.config.Region, r.config.Zone)
		}
		if r.config.Region != "" {
			f = fmt.Sprintf(`%s AND resource.labels.zone = starts_with("%s")`, f, r.config.Region)
		} else if r.config.Zone != "" {
			f = fmt.Sprintf(`%s AND resource.labels.zone = "%s"`, f, r.config.Zone)
		}
	}
	return
}

// Returns a GCP TimeInterval based on the ingestDelay and samplePeriod from ListMetricDescriptor
func getTimeInterval(ingestDelay time.Duration, samplePeriod time.Duration, collectionPeriod duration.Duration) (*monitoringpb.TimeInterval, bool) {
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
		endTime = time.Now().UTC().Add(-ingestDelay)
		startTime = endTime.Add(-samplePeriod)
		needsAggregation = false
	}

	interval := &monitoringpb.TimeInterval{
		StartTime: &timestamp.Timestamp{
			Seconds: startTime.Unix(),
		},
		EndTime: &timestamp.Timestamp{
			Seconds: endTime.Unix(),
		},
	}

	return interval, needsAggregation
}

func constructAggregation(period duration.Duration, perSeriesAligner string, needsAggregation bool) *monitoringpb.Aggregation {
	aligner := "ALIGN_NONE"
	if needsAggregation {
		aligner = perSeriesAligner
		if perSeriesAligner == "" {
			// set to default aggregation ALIGN_MEAN
			aligner = "ALIGN_MEAN"
		}
	}

	aggregation := &monitoringpb.Aggregation{
		PerSeriesAligner: googlecloud.AlignersMapToGCP[aligner],
		AlignmentPeriod:  &period,
	}
	return aggregation
}
