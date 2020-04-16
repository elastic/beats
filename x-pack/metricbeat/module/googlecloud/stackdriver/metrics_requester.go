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

func (r *stackdriverMetricsRequester) Metric(ctx context.Context, m string, timeInterval *monitoringpb.TimeInterval) (out []*monitoringpb.TimeSeries) {
	out = make([]*monitoringpb.TimeSeries, 0)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:     "projects/" + r.config.ProjectID,
		Interval: timeInterval,
		View:     monitoringpb.ListTimeSeriesRequest_FULL,
		Filter:   r.getFilterForMetric(m),
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
			interval := getTimeInterval(metricMeta.ingestDelay, metricMeta.samplePeriod)

			ts := r.Metric(ctx, metricType, interval)

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
func getTimeInterval(ingestDelay time.Duration, samplePeriod time.Duration) *monitoringpb.TimeInterval {
	var startTime, endTime time.Time

	endTime = time.Now().UTC().Add(-ingestDelay * time.Second)
	startTime = endTime.Add(-samplePeriod * time.Second)

	interval := &monitoringpb.TimeInterval{
		StartTime: &timestamp.Timestamp{
			Seconds: startTime.Unix(),
		},
		EndTime: &timestamp.Timestamp{
			Seconds: endTime.Unix(),
		},
	}

	return interval
}
