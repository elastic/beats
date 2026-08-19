// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"github.com/elastic/beats/v9/libbeat/beat"
	"github.com/elastic/beats/v9/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type cloudwatchPoller struct {
	config       config
	region       string
	log          *logp.Logger
	metrics      *inputMetrics
	stateHandler *stateHandler
	status       status.StatusReporter

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
	logGroupId         string
	startTime, endTime time.Time
}

func newCloudwatchPoller(log *logp.Logger, metrics *inputMetrics, awsRegion string, config config, stateHandler *stateHandler, reporter status.StatusReporter) *cloudwatchPoller {
	if metrics == nil {
		metrics = newInputMetrics(monitoring.NewRegistry())
	}

	return &cloudwatchPoller{
		log:                  log,
		metrics:              metrics,
		region:               awsRegion,
		config:               config,
		stateHandler:         stateHandler,
		status:               reporter,
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

func (p *cloudwatchPoller) startWorkers(ctx context.Context, svc *cloudwatchlogs.Client, pipeline beat.Pipeline) error {
	for i := 0; i < p.config.NumberOfWorkers; i++ {
		worker, err := newCWWorker(p.config, p.region, p.metrics, p.status, svc, pipeline, p.log)
		if err != nil {
			return fmt.Errorf("failed to create worker %d: %w", i, err)
		}
		p.workerWg.Add(1)
		go func(wrk *cwWorker) {
			defer p.workerWg.Done()
			wrk.Start(ctx, p.workRequestChan, p.workResponseChan, p.stateHandler)
		}(worker)
	}

	return nil
}

// receive implements the main run loop that distributes tasks to the worker
// goroutines. It accepts a "clock" callback (which on a live input should
// equal time.Now) to allow deterministic unit tests.
func (p *cloudwatchPoller) receive(ctx context.Context, logGroupIDs []string, clock func() time.Time) {
	defer p.workerWg.Wait()

	// startTime and endTime are the bounds of the current scanning interval.
	endTime := clock().Add(-p.config.Latency)

	var startTime time.Time
	// If we're starting at the end of the logs, advance the start time to the most recent scan window
	if p.config.StartPosition == end {
		startTime = endTime.Add(-p.config.ScanFrequency)
	}

	if p.config.StartPosition == beginning {
		startTime = time.Unix(0, 0)
	}

	if p.config.StartPosition == lastSync {
		state, err := p.stateHandler.GetState()
		if err != nil {
			p.log.Errorf("error retrieving state from stateHandler: %v, falling back to %s", err, beginning)
			startTime = time.Unix(0, 0)
		} else {
			startTime = time.UnixMilli(state.LastSyncEpoch)
		}
	}

	for ctx.Err() == nil {
		p.stateHandler.WorkRegister(endTime.UnixMilli(), len(logGroupIDs))

		for _, lg := range logGroupIDs {
			select {
			case <-ctx.Done():
				return
			case <-p.workRequestChan:
				p.workResponseChan <- workResponse{
					logGroupId: lg,
					startTime:  startTime,
					endTime:    endTime,
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
