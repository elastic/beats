// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/statestore"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

type cloudwatchPoller struct {
	numberOfWorkers      int
	apiSleep             time.Duration
	region               string
	logStreams           []string
	logStreamPrefix      string
	startTime            int64
	endTime              int64
	prevEndTime          int64
	workerSem            *awscommon.Sem
	log                  *logp.Logger
	metrics              *inputMetrics
	store                *statestore.Store
	workersListingMap    *sync.Map
	workersProcessingMap *sync.Map
}

func newCloudwatchPoller(log *logp.Logger, metrics *inputMetrics,
	store *statestore.Store,
	awsRegion string, apiSleep time.Duration,
	numberOfWorkers int, logStreams []string, logStreamPrefix string) *cloudwatchPoller {
	if metrics == nil {
		metrics = newInputMetrics(monitoring.NewRegistry(), "")
	}

	return &cloudwatchPoller{
		numberOfWorkers:      numberOfWorkers,
		apiSleep:             apiSleep,
		region:               awsRegion,
		logStreams:           logStreams,
		logStreamPrefix:      logStreamPrefix,
		startTime:            int64(0),
		endTime:              int64(0),
		workerSem:            awscommon.NewSem(numberOfWorkers),
		log:                  log,
		metrics:              metrics,
		store:                store,
		workersListingMap:    new(sync.Map),
		workersProcessingMap: new(sync.Map),
	}
}

func (p *cloudwatchPoller) run(svc cloudwatchlogsiface.ClientAPI, logGroup string, startTime int64, endTime int64, logProcessor *logProcessor) {
	err := p.getLogEventsFromCloudWatch(svc, logGroup, startTime, endTime, logProcessor)
	if err != nil {
		var err *awssdk.RequestCanceledError
		if errors.As(err, &err) {
			p.log.Error("getLogEventsFromCloudWatch failed with RequestCanceledError: ", err)
		}
		p.log.Error("getLogEventsFromCloudWatch failed: ", err)
	}
}

// getLogEventsFromCloudWatch uses FilterLogEvents API to collect logs from CloudWatch
func (p *cloudwatchPoller) getLogEventsFromCloudWatch(svc cloudwatchlogsiface.ClientAPI, logGroup string, startTime int64, endTime int64, logProcessor *logProcessor) error {
	// construct FilterLogEventsInput
	filterLogEventsInput := p.constructFilterLogEventsInput(startTime, endTime, logGroup)

	// make API request
	req := svc.FilterLogEventsRequest(filterLogEventsInput)
	paginator := cloudwatchlogs.NewFilterLogEventsPaginator(req)
	for paginator.Next(context.TODO()) {
		page := paginator.CurrentPage()
		p.metrics.apiCallsTotal.Inc()

		logEvents := page.Events
		p.metrics.logEventsReceivedTotal.Add(uint64(len(logEvents)))

		// This sleep is to avoid hitting the FilterLogEvents API limit(5 transactions per second (TPS)/account/Region).
		p.log.Debugf("sleeping for %v before making FilterLogEvents API call again", p.apiSleep)
		time.Sleep(p.apiSleep)
		p.log.Debug("done sleeping")

		p.log.Debugf("Processing #%v events", len(logEvents))
		err := logProcessor.processLogEvents(logEvents, logGroup, p.region)
		if err != nil {
			err = errors.Wrap(err, "processLogEvents failed")
			p.log.Error(err)
		}
	}

	if err := paginator.Err(); err != nil {
		return errors.Wrap(err, "error FilterLogEvents with Paginator")
	}
	return nil
}

func (p *cloudwatchPoller) constructFilterLogEventsInput(startTime int64, endTime int64, logGroup string) *cloudwatchlogs.FilterLogEventsInput {
	filterLogEventsInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: awssdk.String(logGroup),
<<<<<<< HEAD
		StartTime:    awssdk.Int64(startTime),
		EndTime:      awssdk.Int64(endTime),
=======
		StartTime:    awssdk.Int64(unixMsFromTime(startTime)),
		EndTime:      awssdk.Int64(unixMsFromTime(endTime)),
>>>>>>> c00345ffc1 ([input/awscloudwatch] Set startTime to 0 for the first iteration of retrieving log events from CloudWatch (#40079))
	}

	if len(p.logStreams) > 0 {
		filterLogEventsInput.LogStreamNames = p.logStreams
	}

	if p.logStreamPrefix != "" {
		filterLogEventsInput.LogStreamNamePrefix = awssdk.String(p.logStreamPrefix)
	}
	return filterLogEventsInput
}
<<<<<<< HEAD
=======

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

// unixMsFromTime converts time to unix milliseconds.
// Returns 0 both the init time `time.Time{}`, instead of -6795364578871
func unixMsFromTime(v time.Time) int64 {
	if v.IsZero() {
		return 0
	}
	return v.UnixNano() / int64(time.Millisecond)
}
>>>>>>> c00345ffc1 ([input/awscloudwatch] Set startTime to 0 for the first iteration of retrieving log events from CloudWatch (#40079))
