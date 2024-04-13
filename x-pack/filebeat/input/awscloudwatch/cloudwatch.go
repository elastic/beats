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

	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-libs/logp"
)

type cloudwatchPoller struct {
	numberOfWorkers      int
	apiSleep             time.Duration
	region               string
	logStreams           []*string
	logStreamPrefix      string
	workerSem            *awscommon.Sem
	log                  *logp.Logger
	metrics              *inputMetrics
	workersListingMap    *sync.Map
	workersProcessingMap *sync.Map
}

func newCloudwatchPoller(log *logp.Logger, metrics *inputMetrics,
	awsRegion string, apiSleep time.Duration,
	numberOfWorkers int, logStreams []*string, logStreamPrefix string) *cloudwatchPoller {
	if metrics == nil {
		metrics = newInputMetrics("", nil)
	}

	return &cloudwatchPoller{
		numberOfWorkers:      numberOfWorkers,
		apiSleep:             apiSleep,
		region:               awsRegion,
		logStreams:           logStreams,
		logStreamPrefix:      logStreamPrefix,
		workerSem:            awscommon.NewSem(numberOfWorkers),
		log:                  log,
		metrics:              metrics,
		workersListingMap:    new(sync.Map),
		workersProcessingMap: new(sync.Map),
	}
}

func (p *cloudwatchPoller) run(svc *cloudwatchlogs.Client, logGroup string, startTime int64, endTime int64, logProcessor *logProcessor) {
	err := p.getLogEventsFromCloudWatch(svc, logGroup, startTime, endTime, logProcessor)
	if err != nil {
		var errRequestCanceled *awssdk.RequestCanceledError
		if errors.As(err, &errRequestCanceled) {
			p.log.Error("getLogEventsFromCloudWatch failed with RequestCanceledError: ", err)
		}
		p.log.Error("getLogEventsFromCloudWatch failed: ", err)
	}
}

// getLogEventsFromCloudWatch uses FilterLogEvents API to collect logs from CloudWatch
func (p *cloudwatchPoller) getLogEventsFromCloudWatch(svc *cloudwatchlogs.Client, logGroup string, startTime int64, endTime int64, logProcessor *logProcessor) error {
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
		p.log.Debugf("sleeping for %v before making FilterLogEvents API call again", p.apiSleep)
		time.Sleep(p.apiSleep)
		p.log.Debug("done sleeping")

		p.log.Debugf("Processing #%v events", len(logEvents))
		logProcessor.processLogEvents(logEvents, logGroup, p.region)
	}
	return nil
}

func (p *cloudwatchPoller) constructFilterLogEventsInput(startTime int64, endTime int64, logGroup string) *cloudwatchlogs.FilterLogEventsInput {
	filterLogEventsInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: awssdk.String(logGroup),
		StartTime:    awssdk.Int64(startTime),
		EndTime:      awssdk.Int64(endTime),
	}

	if len(p.logStreams) > 0 {
		for _, stream := range p.logStreams {
			filterLogEventsInput.LogStreamNames = append(filterLogEventsInput.LogStreamNames, *stream)
		}
	}

	if p.logStreamPrefix != "" {
		filterLogEventsInput.LogStreamNamePrefix = awssdk.String(p.logStreamPrefix)
	}
	return filterLogEventsInput
}
