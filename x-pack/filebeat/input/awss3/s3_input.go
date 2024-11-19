// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"fmt"
	"sync"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

// var instead of const so it can be reduced during unit tests (instead of waiting
// through 10 minutes of retry backoff)
var readerLoopMaxCircuitBreaker = 10

type s3PollerInput struct {
	log             *logp.Logger
	pipeline        beat.Pipeline
	config          config
	awsConfig       awssdk.Config
	store           beater.StateStore
	provider        string
	s3              s3API
	metrics         *inputMetrics
	s3ObjectHandler s3ObjectHandlerFactory
	states          *states
}

func newS3PollerInput(
	config config,
	awsConfig awssdk.Config,
	store beater.StateStore,
) (v2.Input, error) {

	return &s3PollerInput{
		config:    config,
		awsConfig: awsConfig,
		store:     store,
	}, nil
}

func (in *s3PollerInput) Name() string { return inputName }

func (in *s3PollerInput) Test(ctx v2.TestContext) error {
	return nil
}

func (in *s3PollerInput) Run(
	inputContext v2.Context,
	pipeline beat.Pipeline,
) error {
	in.log = inputContext.Logger.Named("s3")
	in.pipeline = pipeline
	var err error

	// Load the persistent S3 polling state.
	in.states, err = newStates(in.log, in.store)
	if err != nil {
		return fmt.Errorf("can not start persistent store: %w", err)
	}
	defer in.states.Close()

	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)
	in.s3, err = in.createS3API(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 API: %w", err)
	}

	in.metrics = newInputMetrics(inputContext.ID, nil, in.config.NumberOfWorkers)
	defer in.metrics.Close()

	in.s3ObjectHandler = newS3ObjectProcessorFactory(
		in.metrics,
		in.s3,
		in.config.getFileSelectors(),
		in.config.BackupConfig)

	in.run(ctx)

	return nil
}

func (in *s3PollerInput) run(ctx context.Context) {
	// Scan the bucket in a loop, delaying by the configured interval each
	// iteration.
	for ctx.Err() == nil {
		in.runPoll(ctx)
		_ = timed.Wait(ctx, in.config.BucketListInterval)
	}
}

func (in *s3PollerInput) runPoll(ctx context.Context) {
	var workerWg sync.WaitGroup
	workChan := make(chan state)

	// Start the worker goroutines to listen on the work channel
	for i := 0; i < in.config.NumberOfWorkers; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			in.workerLoop(ctx, workChan)
		}()
	}

	// Start reading data and wait for its processing to be done
	in.readerLoop(ctx, workChan)
	workerWg.Wait()
}

func (in *s3PollerInput) workerLoop(ctx context.Context, workChan <-chan state) {
	acks := newAWSACKHandler()
	// Create client for publishing events and receive notification of their ACKs.
	client, err := createPipelineClient(in.pipeline, acks)
	if err != nil {
		in.log.Errorf("failed to create pipeline client: %v", err.Error())
		return
	}
	defer client.Close()
	defer acks.Close()

	rateLimitWaiter := backoff.NewEqualJitterBackoff(ctx.Done(), 1, 120)

	for _state := range workChan {
		state := _state
		event := in.s3EventForState(state)

		objHandler := in.s3ObjectHandler.Create(ctx, event)
		if objHandler == nil {
			in.log.Debugw("empty s3 processor (no matching reader configs).", "state", state)
			continue
		}

		// Process S3 object (download, parse, create events).
		publishCount := 0
		err := objHandler.ProcessS3Object(in.log, func(e beat.Event) {
			in.metrics.s3EventsCreatedTotal.Inc()
			client.Publish(e)
			publishCount++
		})
		in.metrics.s3EventsPerObject.Update(int64(publishCount))
		if errors.Is(err, errS3DownloadFailed) {
			// Download errors are ephemeral. Add a backoff delay, then skip to the
			// next iteration so we don't mark the object as permanently failed.
			rateLimitWaiter.Wait()
			continue
		}
		// Reset the rate limit delay on results that aren't download errors.
		rateLimitWaiter.Reset()

		// Update state, but don't persist it until this object is acknowledged.
		if err != nil {
			in.log.Errorf("failed processing S3 event for object key %q in bucket %q: %v",
				state.Key, state.Bucket, err.Error())

			// Non-retryable error.
			state.Failed = true
		} else {
			state.Stored = true
		}

		// Add the cleanup handling to the acks helper
		acks.Add(publishCount, func() {
			err := in.states.AddState(state)
			if err != nil {
				in.log.Errorf("saving completed object state: %v", err.Error())
			}

			// Metrics
			in.metrics.s3ObjectsAckedTotal.Inc()
		})
	}
}

func (in *s3PollerInput) readerLoop(ctx context.Context, workChan chan<- state) {
	defer close(workChan)

	bucketName := getBucketNameFromARN(in.config.getBucketARN())

	errorBackoff := backoff.NewEqualJitterBackoff(ctx.Done(), 1, 120)
	circuitBreaker := 0
	paginator := in.s3.ListObjectsPaginator(bucketName, in.config.BucketListPrefix)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)

		if err != nil {
			in.log.Warnw("Error when paginating listing.", "error", err)
			// QuotaExceededError is client-side rate limiting in the AWS sdk,
			// don't include it in the circuit breaker count
			if !errors.As(err, &ratelimit.QuotaExceededError{}) {
				circuitBreaker++
				if circuitBreaker >= readerLoopMaxCircuitBreaker {
					in.log.Warnw(fmt.Sprintf("%d consecutive error when paginating listing, breaking the circuit.", circuitBreaker), "error", err)
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
		in.metrics.s3ObjectsListedTotal.Add(uint64(totListedObjects))
		for _, object := range page.Contents {
			state := newState(bucketName, *object.Key, *object.ETag, *object.LastModified)
			if in.states.IsProcessed(state) {
				in.log.Debugw("skipping state.", "state", state)
				continue
			}

			workChan <- state

			in.metrics.s3ObjectsProcessedTotal.Inc()
		}
	}
}

func (in *s3PollerInput) s3EventForState(state state) s3EventV2 {
	event := s3EventV2{}
	event.AWSRegion = in.awsConfig.Region
	event.Provider = in.provider
	event.S3.Bucket.Name = state.Bucket
	event.S3.Bucket.ARN = in.config.getBucketARN()
	event.S3.Object.Key = state.Key
	return event
}
