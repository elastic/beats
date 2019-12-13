// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stackdriver

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/logp"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/beats/x-pack/metricbeat/module/googlecloud"
)

func newStackdriverMetricsRequester(ctx context.Context, c config, window time.Duration, logger *logp.Logger) (*stackdriverMetricsRequester, error) {
	interval, err := getTimeInterval(window)
	if err != nil {
		return nil, errors.Wrap(err, "error trying to get time window")
	}

	client, err := monitoring.NewMetricClient(ctx, c.opt)
	if err != nil {
		return nil, errors.Wrap(err, "error creating Stackdriver client")
	}

	return &stackdriverMetricsRequester{
		config:   c,
		client:   client,
		logger:   logger,
		interval: interval,
	}, nil
}

type stackdriverMetricsRequester struct {
	config config

	client   *monitoring.MetricClient
	interval *monitoringpb.TimeInterval

	logger *logp.Logger
}

func (r *stackdriverMetricsRequester) Metric(ctx context.Context, m string) (out []*monitoringpb.TimeSeries) {
	out = make([]*monitoringpb.TimeSeries, 0)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:     "projects/" + r.config.ProjectID,
		Interval: r.interval,
		View:     monitoringpb.ListTimeSeriesRequest_FULL,
		Filter:   fmt.Sprintf(`metric.type="%s" AND resource.labels.zone = "%s"`, m, r.config.Zone),
	}

	it := r.client.ListTimeSeries(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			r.logger.Error(errors.Wrapf(err, "could not read time series value: %v", m))
			break
		}

		out = append(out, resp)
	}

	return
}

func (r *stackdriverMetricsRequester) Metrics(ctx context.Context, ms []string) ([]*monitoringpb.TimeSeries, error) {
	lock := sync.Mutex{}
	var wg sync.WaitGroup
	results := make([]*monitoringpb.TimeSeries, 0)

	for _, metric := range ms {
		wg.Add(1)

		go func(m string) {
			defer wg.Done()

			ts := r.Metric(ctx, m)

			for _, timeSeries := range ts {
				func(ts *monitoringpb.TimeSeries) {
					defer lock.Unlock()
					lock.Lock()
					results = append(results, timeSeries)
				}(timeSeries)
			}
		}(metric)
	}

	wg.Wait()

	if len(results) == 0 {
		return nil, errors.New("service returned 0 metrics")
	}

	return results, nil
}

// Returns a GCP TimeInterval based on the provided config
func getTimeInterval(windowTime time.Duration) (*monitoringpb.TimeInterval, error) {
	var startTime, endTime time.Time

	if windowTime > 0 {
		endTime = time.Now().UTC()
		startTime = time.Now().UTC().Add(-windowTime)
	} else if startTime.IsZero() && endTime.IsZero() {
		return nil, errors.New("a window time or start and end time must be provided to create a time window to fetch metrics from")
	} else if windowTime.Minutes() > googlecloud.MaxTimeIntervalDataWindowMinutes {
		return nil, errors.Errorf("the provided window time is too big. No more than %d minutes can be fetched", googlecloud.MaxTimeIntervalDataWindowMinutes)
	} else if startTime.Unix()-endTime.Unix() >= 0 {
		return nil, errors.New("start time cannot be later than or equal than end time")
	}

	interval := &monitoringpb.TimeInterval{
		StartTime: &timestamp.Timestamp{
			Seconds: startTime.Unix(),
		},
		EndTime: &timestamp.Timestamp{
			Seconds: endTime.Unix(),
		},
	}

	return interval, nil
}
