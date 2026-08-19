// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/smithy-go/transport/http"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"github.com/elastic/beats/v9/libbeat/beat"
	"github.com/elastic/beats/v9/libbeat/common/acker"
	"github.com/elastic/beats/v9/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
)

type cwWorker struct {
	client    beat.Client
	config    config
	log       *logp.Logger
	metrics   *inputMetrics
	processor *logProcessor
	region    string
	status    status.StatusReporter
	svc       *cloudwatchlogs.Client
	tracker   *ackTracker
}

func newCWWorker(cfg config,
	region string,
	metrics *inputMetrics,
	status status.StatusReporter,
	svc *cloudwatchlogs.Client,
	pipeline beat.Pipeline,
	log *logp.Logger) (*cwWorker, error) {

	cw := &cwWorker{
		config:  cfg,
		region:  region,
		metrics: metrics,
		status:  status,
		svc:     svc,
		log:     log,
	}

	tracker := newACKTracker()
	client, err := pipeline.ConnectWith(beat.ClientConfig{
		EventListener: acker.TrackingCounter(func(_ int, by int) {
			tracker.increaseAck(by)
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline client: %w", err)
	}

	cw.client = client
	cw.processor = newLogProcessor(log, metrics, client)
	cw.tracker = tracker
	return cw, nil
}

// Start the CloudWatch worker that requests and wait for work. Contains blocking operations, hence must be called concurrently.
func (w *cwWorker) Start(ctx context.Context, workReq chan struct{}, workRsp chan workResponse, handler *stateHandler) {
	defer w.client.Close()
	defer w.tracker.close()

	for {
		var work workResponse
		select {
		case <-ctx.Done():
			return
		case workReq <- struct{}{}:
			work = <-workRsp
		}

		w.log.Infof("aws-cloudwatch input worker for log group: '%v' has started", work.logGroupId)
		workedCount := w.run(ctx, work.logGroupId, work.startTime, work.endTime)
		w.log.Infof("aws-cloudwatch input worker for log group '%v' has completed.", work.logGroupId)

		select {
		case <-ctx.Done():
			w.log.Debugf("context completed before acknowledging delivery for log group '%v'", work.logGroupId)
		case <-w.tracker.waitFor(workedCount):
			handler.WorkComplete(work.endTime.UnixMilli())
			w.log.Debugf("all events (%d) acknowledged for log group '%v'", workedCount, work.logGroupId)
		}
	}
}

func (w *cwWorker) run(ctx context.Context, logGroupId string, startTime, endTime time.Time) int {
	count, err := w.getLogEventsFromCloudWatch(ctx, logGroupId, startTime, endTime)
	if err == nil {
		// return fast for non-errors
		w.status.UpdateStatus(status.Running, "Input is running")
		return count
	}

	// handle errors
	var errRequestCanceled *awssdk.RequestCanceledError
	if errors.As(err, &errRequestCanceled) {
		w.log.Error("getLogEventsFromCloudWatch failed with RequestCanceledError: ", errRequestCanceled)
	} else {
		w.log.Error("getLogEventsFromCloudWatch failed: ", err)
	}

	var rspError *http.ResponseError
	if errors.As(err, &rspError) && rspError.Response != nil {
		// update status with context details if Response is available
		w.status.UpdateStatus(status.Degraded, fmt.Sprintf("Log group listing failed, status: %d, error: %s", rspError.Response.StatusCode, rspError.Error()))
		return count
	}

	w.status.UpdateStatus(status.Degraded, fmt.Sprintf("Log group listing failed, error: %s", err.Error()))
	return count
}

// getLogEventsFromCloudWatch uses FilterLogEvents API to collect logs from CloudWatch
func (w *cwWorker) getLogEventsFromCloudWatch(ctx context.Context, logGroupId string, startTime, endTime time.Time) (int, error) {
	var logCount int
	// construct FilterLogEventsInput
	filterLogEventsInput := w.constructFilterLogEventsInput(startTime, endTime, logGroupId)
	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(w.svc, filterLogEventsInput)
	for paginator.HasMorePages() && ctx.Err() == nil {
		filterLogEventsOutput, err := paginator.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("error FilterLogEvents with Paginator: %w", err)
		}

		w.metrics.apiCallsTotal.Inc()
		logEvents := filterLogEventsOutput.Events
		w.metrics.logEventsReceivedTotal.Add(uint64(len(logEvents)))

		// This sleep is to avoid hitting the FilterLogEvents API limit(5 transactions per second (TPS)/account/Region).
		w.log.Debugf("sleeping for %v before making FilterLogEvents API call again", w.config.APISleep)
		time.Sleep(w.config.APISleep)
		w.log.Debug("done sleeping")

		w.log.Debugf("Processing #%v events", len(logEvents))
		w.processor.processLogEvents(logEvents, logGroupId, w.region)
		logCount += len(logEvents)
	}

	return logCount, nil
}

func (w *cwWorker) constructFilterLogEventsInput(startTime, endTime time.Time, logGroupId string) *cloudwatchlogs.FilterLogEventsInput {
	w.log.Debugf("FilterLogEventsInput for log group: '%s' with startTime = '%v' and endTime = '%v'", logGroupId, unixMsFromTime(startTime), unixMsFromTime(endTime))
	filterLogEventsInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupIdentifier: awssdk.String(logGroupId),
		StartTime:          awssdk.Int64(unixMsFromTime(startTime)),
		EndTime:            awssdk.Int64(unixMsFromTime(endTime)),
	}

	if len(w.config.LogStreams) > 0 {
		for _, stream := range w.config.LogStreams {
			filterLogEventsInput.LogStreamNames = append(filterLogEventsInput.LogStreamNames, *stream)
		}
	}

	if w.config.LogStreamPrefix != "" {
		filterLogEventsInput.LogStreamNamePrefix = awssdk.String(w.config.LogStreamPrefix)
	}
	return filterLogEventsInput
}

// ackTracker tracks acknowledgements of an individual worker
type ackTracker struct {
	increment  chan int
	checkTotal chan int
	// Emits tracking completion once total and increment count matches
	complete chan struct{}
	// Chan responsible to distribute shutdown signal.
	shutdown chan struct{}
}

func newACKTracker() *ackTracker {
	tracker := &ackTracker{
		increment:  make(chan int, 1),
		checkTotal: make(chan int, 1),
		complete:   make(chan struct{}, 1),
		shutdown:   make(chan struct{}),
	}

	go tracker.runner()

	return tracker
}

func (ac *ackTracker) close() {
	close(ac.shutdown)
}

// increaseAck allows to increase acknowledged count.
// See runner for internal work.
func (ac *ackTracker) increaseAck(by int) {
	select {
	case ac.increment <- by:
	case <-ac.shutdown: // Make sure to not block during a shutdown
	}
}

// waitFor accepts a total value to be completed where completion will be communicated through returned channel.
// See runner for internal work.
func (ac *ackTracker) waitFor(total int) <-chan struct{} {
	ac.checkTotal <- total

	return ac.complete
}

// runner contains blocking calls and tracks acknowledgements and totals.
// If acknowledgements reach total, it emits complete signal.
// Acknowledgements can be taken for zero total value.
func (ac *ackTracker) runner() {
	total := -1
	count := 0
	for {
		select {
		case <-ac.shutdown:
			return
		case t := <-ac.checkTotal:
			total = t
		case c := <-ac.increment:
			count += c
		}

		if total >= 0 && count >= total {
			count -= total
			total = -1
			ac.complete <- struct{}{}
		}
	}
}
