// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/duration"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
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

func (r *metricsRequester) Metric(ctx context.Context, metricType string, timeInterval *monitoringpb.TimeInterval, aligner string) (out timeSeriesWithAligner) {
	timeSeries := make([]*monitoringpb.TimeSeries, 0)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:     "projects/" + r.config.ProjectID,
		Interval: timeInterval,
		View:     monitoringpb.ListTimeSeriesRequest_FULL,
		Filter:   r.getFilterForMetric(metricType),
		Aggregation: &monitoringpb.Aggregation{
			PerSeriesAligner: gcp.AlignersMapToGCP[aligner],
			AlignmentPeriod:  r.config.period,
		},
	}

	it := r.client.ListTimeSeries(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			r.logger.Errorf("Could not read time series value: %s: %v", metricType, err)
			break
		}

		timeSeries = append(timeSeries, resp)
	}

	out.aligner = aligner
	out.timeSeries = timeSeries
	return
}

func (r *metricsRequester) Metrics(ctx context.Context, sdc metricsConfig, metricsMeta map[string]metricMeta) ([]timeSeriesWithAligner, error) {
	var lock sync.Mutex
	var wg sync.WaitGroup
	results := make([]timeSeriesWithAligner, 0)

	aligner := sdc.Aligner
	for mt, meta := range metricsMeta {
		wg.Add(1)

		metricMeta := meta
		go func(mt string) {
			defer wg.Done()

			r.logger.Debugf("For metricType %s, metricMeta = %d", mt, metricMeta)
			interval, aligner := getTimeIntervalAligner(metricMeta.ingestDelay, metricMeta.samplePeriod, r.config.period, aligner)
			ts := r.Metric(ctx, mt, interval, aligner)
			lock.Lock()
			defer lock.Unlock()
			results = append(results, ts)
		}(mt)
	}

	wg.Wait()
	return results, nil
}

var serviceRegexp = regexp.MustCompile(`^(?P<service>[a-z]+)\.googleapis.com.*`)

// getFilterForMetric returns the filter associated with the corresponding filter. Some services like Pub/Sub fails
// if they have a region specified.
func (r *metricsRequester) getFilterForMetric(m string) (f string) {
	f = fmt.Sprintf(`metric.type="%s"`, m)
	if r.config.Zone == "" && r.config.Region == "" {
		return
	}

	service := serviceRegexp.ReplaceAllString(m, "${service}")

	switch service {
	case gcp.ServicePubsub, gcp.ServiceLoadBalancing, gcp.ServiceCloudFunctions:
		return
	case gcp.ServiceStorage:
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
			region := r.config.Region
			if strings.HasSuffix(r.config.Region, "*") {
				region = strings.TrimSuffix(r.config.Region, "*")
			}
			f = fmt.Sprintf(`%s AND resource.labels.zone = starts_with("%s")`, f, region)
		} else if r.config.Zone != "" {
			zone := r.config.Zone
			if strings.HasSuffix(r.config.Zone, "*") {
				zone = strings.TrimSuffix(r.config.Zone, "*")
			}
			f = fmt.Sprintf(`%s AND resource.labels.zone = starts_with("%s")`, f, zone)
		}
	}
	r.logger.Debugf("ListTimeSeries API filter = %s", f)
	return
}

// Returns a GCP TimeInterval based on the ingestDelay and samplePeriod from ListMetricDescriptor
func getTimeIntervalAligner(ingestDelay time.Duration, samplePeriod time.Duration, collectionPeriod *duration.Duration, inputAligner string) (*monitoringpb.TimeInterval, string) {
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
		StartTime: &timestamp.Timestamp{
			Seconds: startTime.Unix(),
		},
		EndTime: &timestamp.Timestamp{
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
