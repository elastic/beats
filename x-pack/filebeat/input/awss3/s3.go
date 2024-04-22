// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

const maxCircuitBreaker = 10

type commitWriteState struct {
	time.Time
}

type s3ObjectPayload struct {
	s3ObjectHandler s3ObjectHandler
	s3ObjectEvent   s3EventV2
	objectState     state
}

type s3Poller struct {
	numberOfWorkers      int
	bucket               string
	listPrefix           string
	region               string
	provider             string
	bucketPollInterval   time.Duration
	workerSem            *awscommon.Sem
	s3                   s3API
	log                  *logp.Logger
	metrics              *inputMetrics
	client               beat.Client
	s3ObjectHandler      s3ObjectHandlerFactory
	states               *states
	workersProcessingMap *sync.Map
}

func newS3Poller(log *logp.Logger,
	metrics *inputMetrics,
	s3 s3API,
	client beat.Client,
	s3ObjectHandler s3ObjectHandlerFactory,
	states *states,
	bucket string,
	listPrefix string,
	awsRegion string,
	provider string,
	numberOfWorkers int,
	bucketPollInterval time.Duration,
) *s3Poller {
	if metrics == nil {
		// Metrics are optional. Initialize a stub.
		metrics = newInputMetrics("", nil, 0)
	}
	return &s3Poller{
		numberOfWorkers:      numberOfWorkers,
		bucket:               bucket,
		listPrefix:           listPrefix,
		region:               awsRegion,
		provider:             provider,
		bucketPollInterval:   bucketPollInterval,
		workerSem:            awscommon.NewSem(numberOfWorkers),
		s3:                   s3,
		log:                  log,
		metrics:              metrics,
		client:               client,
		s3ObjectHandler:      s3ObjectHandler,
		states:               states,
		workersProcessingMap: new(sync.Map),
	}
}

func (p *s3Poller) createS3ObjectProcessor(ctx context.Context, state state) s3ObjectHandler {
	event := s3EventV2{}
	event.AWSRegion = p.region
	event.Provider = p.provider
	event.S3.Bucket.Name = state.Bucket
	event.S3.Bucket.ARN = p.bucket
	event.S3.Object.Key = state.Key

	acker := awscommon.NewEventACKTracker(ctx)

	return p.s3ObjectHandler.Create(ctx, p.log, p.client, acker, event)
}

func (p *s3Poller) workerLoop(ctx context.Context, s3ObjectPayloadChan <-chan *s3ObjectPayload) {
	rateLimitWaiter := backoff.NewEqualJitterBackoff(ctx.Done(), 1, 120)

	for s3ObjectPayload := range s3ObjectPayloadChan {
		objHandler := s3ObjectPayload.s3ObjectHandler
		state := s3ObjectPayload.objectState

		// Process S3 object (download, parse, create events).
		err := objHandler.ProcessS3Object()
		if errors.Is(err, s3DownloadError) {
			// Download errors are ephemeral. Add a backoff delay, then skip to the
			// next iteration so we don't mark the object as permanently failed.
			rateLimitWaiter.Wait()
			continue
		}
		// Reset the rate limit delay on results that aren't download errors.
		rateLimitWaiter.Reset()

		// Wait for downloaded objects to be ACKed.
		objHandler.Wait()

		if err != nil {
			p.log.Errorf("failed processing S3 event for object key %q in bucket %q: %v",
				state.Key, state.Bucket, err.Error())

			// Non-retryable error.
			state.Failed = true
		} else {
			state.Stored = true
		}

		// Persist the result
		p.states.AddState(state)

		// Metrics
		p.metrics.s3ObjectsAckedTotal.Inc()
	}
}

func (p *s3Poller) readerLoop(ctx context.Context, s3ObjectPayloadChan chan<- *s3ObjectPayload) {
	defer close(s3ObjectPayloadChan)

	bucketName := getBucketNameFromARN(p.bucket)

	errorBackoff := backoff.NewEqualJitterBackoff(ctx.Done(), 1, 120)
	circuitBreaker := 0
	paginator := p.s3.ListObjectsPaginator(bucketName, p.listPrefix)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)

		if err != nil {
			if !paginator.HasMorePages() {
				break
			}

			p.log.Warnw("Error when paginating listing.", "error", err)
			// QuotaExceededError is client-side rate limiting in the AWS sdk,
			// don't include it in the circuit breaker count
			if !errors.As(err, &ratelimit.QuotaExceededError{}) {
				circuitBreaker++
				if circuitBreaker >= maxCircuitBreaker {
					p.log.Warnw(fmt.Sprintf("%d consecutive error when paginating listing, breaking the circuit.", circuitBreaker), "error", err)
					break
				}
			}
			// add a backoff delay and try again
			errorBackoff.Wait()
			continue
		}
		// Reset the circuit breaker and the error backoff if a read is successful
		circuitBreaker = 0
		errorBackoff.Reset()

		totListedObjects := len(page.Contents)

		// Metrics
		p.metrics.s3ObjectsListedTotal.Add(uint64(totListedObjects))
		for _, object := range page.Contents {
			state := newState(bucketName, *object.Key, *object.ETag, *object.LastModified)
			if p.states.AlreadyProcessed(state) {
				p.log.Debugw("skipping state.", "state", state)
				continue
			}

			s3Processor := p.createS3ObjectProcessor(ctx, state)
			if s3Processor == nil {
				p.log.Debugw("empty s3 processor.", "state", state)
				continue
			}

			s3ObjectPayloadChan <- &s3ObjectPayload{
				s3ObjectHandler: s3Processor,
				objectState:     state,
			}

			p.metrics.s3ObjectsProcessedTotal.Inc()
		}
	}
}

func (p *s3Poller) Poll(ctx context.Context) error {
	for ctx.Err() == nil {
		var workerWg sync.WaitGroup
		workChan := make(chan *s3ObjectPayload)

		// Start the worker goroutines to listen on the work channel
		for i := 0; i < p.numberOfWorkers; i++ {
			workerWg.Add(1)
			go func() {
				defer workerWg.Done()
				p.workerLoop(ctx, workChan)
			}()
		}

		// Start reading data and wait for its processing to be done
		p.readerLoop(ctx, workChan)
		workerWg.Wait()

		_ = timed.Wait(ctx, p.bucketPollInterval)
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		// A canceled context is a normal shutdown.
		return nil
	}
	return ctx.Err()
}
