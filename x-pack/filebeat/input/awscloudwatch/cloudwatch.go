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
		StartTime:    awssdk.Int64(startTime),
		EndTime:      awssdk.Int64(endTime),
		Limit:        awssdk.Int64(100),
	}

	if len(p.logStreams) > 0 {
		filterLogEventsInput.LogStreamNames = p.logStreams
	}

	if p.logStreamPrefix != "" {
		filterLogEventsInput.LogStreamNamePrefix = awssdk.String(p.logStreamPrefix)
	}
	return filterLogEventsInput
}
