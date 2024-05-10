// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"github.com/elastic/elastic-agent-libs/logp"
)

type cloudwatchPoller struct {
	config               config
	region               string
	log                  *logp.Logger
	metrics              *inputMetrics
	workersListingMap    *sync.Map
	workersProcessingMap *sync.Map

	// When a worker is ready for its next task, it should
	// send to workRequestChan and then read from workResponseChan.
	// The worker can cancel the request based on other context
	// cancellations, but if the write succeeds it _must_ read from
	// workResponseChan to avoid deadlocking the main loop.
	workRequestChan  chan struct{}
	workResponseChan chan workResponse

	workerWg sync.WaitGroup
}

type workResponse struct {
	logGroup           string
	startTime, endTime time.Time
}

func newCloudwatchPoller(log *logp.Logger, metrics *inputMetrics,
	awsRegion string, config config) *cloudwatchPoller {
	if metrics == nil {
		metrics = newInputMetrics("", nil)
	}

	return &cloudwatchPoller{
		log:                  log,
		metrics:              metrics,
		region:               awsRegion,
		config:               config,
		workersListingMap:    new(sync.Map),
		workersProcessingMap: new(sync.Map),
		// workRequestChan is unbuffered to guarantee that
		// the worker and main loop agree whether a request
		// was sent. workerResponseChan is buffered so the
		// main loop doesn't have to block on the workers
		// while distributing new data.
		workRequestChan:  make(chan struct{}),
		workResponseChan: make(chan workResponse, 10),
	}
}

func (p *cloudwatchPoller) run(svc *cloudwatchlogs.Client, logGroup string, startTime, endTime time.Time, logProcessor *logProcessor) {
	err := p.getLogEventsFromCloudWatch(svc, logGroup, startTime, endTime, logProcessor)
	if err != nil {
		var errRequestCanceled *awssdk.RequestCanceledError
		if errors.As(err, &errRequestCanceled) {
			p.log.Error("getLogEventsFromCloudWatch failed with RequestCanceledError: ", errRequestCanceled)
		}
		p.log.Error("getLogEventsFromCloudWatch failed: ", err)
	}
}

// getLogEventsFromCloudWatch uses FilterLogEvents API to collect logs from CloudWatch
func (p *cloudwatchPoller) getLogEventsFromCloudWatch(svc *cloudwatchlogs.Client, logGroup string, startTime, endTime time.Time, logProcessor *logProcessor) error {
	// construct FilterLogEventsInput
	filterLogEventsInput := p.constructFilterLogEventsInput(startTime, endTime, logGroup)
	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(svc, filterLogEventsInput)
	for paginator.HasMorePages() {
		filterLogEventsOutput, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("error FilterLogEvents with Paginator: %w", err)
		}

		p.metrics.apiCallsTotal.Inc()
		logEvents := filterLogEventsOutput.Events
		p.metrics.logEventsReceivedTotal.Add(uint64(len(logEvents)))

		// This sleep is to avoid hitting the FilterLogEvents API limit(5 transactions per second (TPS)/account/Region).
		p.log.Debugf("sleeping for %v before making FilterLogEvents API call again", p.config.APISleep)
		time.Sleep(p.config.APISleep)
		p.log.Debug("done sleeping")

		p.log.Debugf("Processing #%v events", len(logEvents))
		logProcessor.processLogEvents(logEvents, logGroup, p.region)
	}
	return nil
}

func (p *cloudwatchPoller) constructFilterLogEventsInput(startTime, endTime time.Time, logGroup string) *cloudwatchlogs.FilterLogEventsInput {
	filterLogEventsInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: awssdk.String(logGroup),
		StartTime:    awssdk.Int64(startTime.UnixNano() / int64(time.Millisecond)),
		EndTime:      awssdk.Int64(endTime.UnixNano() / int64(time.Millisecond)),
	}

	if len(p.config.LogStreams) > 0 {
		for _, stream := range p.config.LogStreams {
			filterLogEventsInput.LogStreamNames = append(filterLogEventsInput.LogStreamNames, *stream)
		}
	}

	if p.config.LogStreamPrefix != "" {
		filterLogEventsInput.LogStreamNamePrefix = awssdk.String(p.config.LogStreamPrefix)
	}
	return filterLogEventsInput
}

func (p *cloudwatchPoller) startWorkers(
	ctx context.Context,
	svc *cloudwatchlogs.Client,
	logProcessor *logProcessor,
) {
	for i := 0; i < p.config.NumberOfWorkers; i++ {
		p.workerWg.Add(1)
		go func() {
			defer p.workerWg.Done()
			for {
				var work workResponse
				select {
				case <-ctx.Done():
					return
				case p.workRequestChan <- struct{}{}:
					work = <-p.workResponseChan
				}

				p.log.Infof("aws-cloudwatch input worker for log group: '%v' has started", work.logGroup)
				p.run(svc, work.logGroup, work.startTime, work.endTime, logProcessor)
				p.log.Infof("aws-cloudwatch input worker for log group '%v' has stopped.", work.logGroup)
			}
		}()
	}
}

// receive implements the main run loop that distributes tasks to the worker
// goroutines. It accepts a "clock" callback (which on a live input should
// equal time.Now) to allow deterministic unit tests.
func (p *cloudwatchPoller) receive(ctx context.Context, logGroupNames []string, clock func() time.Time) {
	defer p.workerWg.Wait()
	// startTime and endTime are the bounds of the current scanning interval.
	// If we're starting at the end of the logs, advance the start time to the
	// most recent scan window
	var startTime time.Time
	endTime := clock().Add(-p.config.Latency)
	if p.config.StartPosition == "end" {
		startTime = endTime.Add(-p.config.ScanFrequency)
	}
	for ctx.Err() == nil {
		for _, lg := range logGroupNames {
			select {
			case <-ctx.Done():
				return
			case <-p.workRequestChan:
				p.workResponseChan <- workResponse{
					logGroup:  lg,
					startTime: startTime,
					endTime:   endTime,
				}
			}
		}

		// Delay for ScanFrequency after finishing a time span
		p.log.Debugf("sleeping for %v before checking new logs", p.config.ScanFrequency)
		select {
		case <-time.After(p.config.ScanFrequency):
		case <-ctx.Done():
		}
		p.log.Debug("done sleeping")

		// Advance to the next time span
		startTime, endTime = endTime, clock().Add(-p.config.Latency)
	}
}
