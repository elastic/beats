// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
)

type sqsReaderInput struct {
	config     config
	awsConfig  awssdk.Config
	sqs        sqsAPI
	s3         s3API
	msgHandler sqsProcessor
	log        *logp.Logger
	metrics    *inputMetrics

	// The Beats pipeline, used to create clients for event publication when
	// creating the worker goroutines.
	pipeline beat.Pipeline

	// The expected region based on the queue URL
	detectedRegion string

	// Workers send on workRequestChan to indicate they're ready for the next
	// message, and the reader loop replies on workResponseChan.
	workRequestChan  chan struct{}
	workResponseChan chan types.Message

	// workerWg is used to wait on worker goroutines during shutdown
	workerWg sync.WaitGroup
}

// Simple wrapper to handle creation of internal channels
func newSQSReaderInput(config config, awsConfig awssdk.Config) *sqsReaderInput {
	return &sqsReaderInput{
		config:           config,
		awsConfig:        awsConfig,
		workRequestChan:  make(chan struct{}, config.NumberOfWorkers),
		workResponseChan: make(chan types.Message),
	}
}

func (in *sqsReaderInput) Name() string { return inputName }

func (in *sqsReaderInput) Test(ctx v2.TestContext) error {
	return nil
}

func (in *sqsReaderInput) Run(
	inputContext v2.Context,
	pipeline beat.Pipeline,
) error {
	// Initialize everything for this run
	err := in.setup(inputContext, pipeline)
	if err != nil {
		return err
	}

	// Start the main run loop
	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)
	in.run(ctx)
	in.cleanup()

	return nil
}

// Apply internal initialization based on the parameters of Run, in
// preparation for calling run. setup and run are separate functions so
// tests can apply mocks and overrides before the run loop.
func (in *sqsReaderInput) setup(
	inputContext v2.Context,
	pipeline beat.Pipeline,
) error {
	in.log = inputContext.Logger.With("queue_url", in.config.QueueURL)
	in.pipeline = pipeline

	in.detectedRegion = getRegionFromQueueURL(in.config.QueueURL)
	if in.config.RegionName != "" {
		// Configured region always takes precedence
		in.awsConfig.Region = in.config.RegionName
	} else if in.detectedRegion != "" {
		// Only use detected region if there is no explicit region configured.
		in.awsConfig.Region = in.detectedRegion
	} else if in.config.AWSConfig.DefaultRegion != "" {
		// If we can't find anything else, fall back on the default.
		in.awsConfig.Region = in.config.AWSConfig.DefaultRegion
	} else {
		// If we can't find a usable region, return an error
		return fmt.Errorf("region not specified and failed to get AWS region from queue_url: %w", errBadQueueURL)
	}

	in.sqs = &awsSQSAPI{
		client: sqs.NewFromConfig(in.awsConfig, in.config.sqsConfigModifier),

		queueURL:          in.config.QueueURL,
		apiTimeout:        in.config.APITimeout,
		visibilityTimeout: in.config.VisibilityTimeout,
		longPollWaitTime:  in.config.SQSWaitTime,
	}

	in.s3 = newAWSs3API(s3.NewFromConfig(in.awsConfig, in.config.s3ConfigModifier))

	in.metrics = newInputMetrics(inputContext.ID, nil, in.config.NumberOfWorkers)

	var err error
	in.msgHandler, err = in.createEventProcessor()
	if err != nil {
		return fmt.Errorf("failed to initialize sqs reader: %w", err)
	}
	return nil
}

// Release internal resources created during setup (currently just metrics).
// This is its own function so tests can handle the run loop in isolation.
func (in *sqsReaderInput) cleanup() {
	if in.metrics != nil {
		in.metrics.Close()
	}
}

// Create the main goroutines for the input (workers, message count monitor)
// and begin the run loop.
func (in *sqsReaderInput) run(ctx context.Context) {
	in.logConfigSummary()

	// If we have received a batch, allow SQS grace time to collect
	// and process all the messages before we respect the parent
	// context. This is used only for processing and publication,
	// not for the reader loop, otherwise a new collection could
	// start, and we are back where we started.
	graceCtx := ctx
	if in.config.SQSGraceTime > 0 {
		var cancel context.CancelFunc
		graceCtx, cancel = cancelWithGrace(ctx, in.config.SQSGraceTime)
		defer cancel()
	}

	// Poll metrics periodically in the background.
	//
	// Use the graceCtx here also to ensure that all metrics have
	// been collected; this is much less likely to be an issue.
	go messageCountMonitor{
		sqs:     in.sqs,
		metrics: in.metrics,
	}.run(graceCtx)

	in.startWorkers(ctx, graceCtx)
	in.readerLoop(ctx)

	in.workerWg.Wait()
}

// cancelWithGrace provides a context.Context that will be cancelled by a call
// to the parent's cancellation, but with a delayed timeout. The returned cancel
// function should be called when the returned context.Context is no longer
// needed.
func cancelWithGrace(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.WithoutCancel(parent))
	stop := context.AfterFunc(parent, func() {
		time.AfterFunc(timeout, cancel)
	})
	return ctx, func() {
		stop()
		cancel()
	}
}

func (in *sqsReaderInput) readerLoop(ctx context.Context) {
	// requestCount is the number of outstanding work requests that the
	// reader will try to fulfill
	requestCount := 0
	for ctx.Err() == nil {
		// Block to wait for more requests if requestCount is zero
		requestCount += channelRequestCount(ctx, in.workRequestChan, requestCount == 0)

		msgs := readSQSMessages(ctx, in.log, in.sqs, in.metrics, requestCount)

		for _, msg := range msgs {
			select {
			case <-ctx.Done():
				return
			case in.workResponseChan <- msg:
				requestCount--
			}
		}
	}
}

type sqsWorker struct {
	input      *sqsReaderInput
	client     beat.Client
	ackHandler *awsACKHandler
	// wg is shared with the owning sqsReaderInput. It
	// is incremented prior to the call to newSQSWorker
	// and must be Done either in the unhappy path in
	// that function, or after completion of the work
	// loop.
	wg      *sync.WaitGroup
	pending atomic.Int64
}

func (in *sqsReaderInput) newSQSWorker() (*sqsWorker, error) {
	// Create a pipeline client scoped to this worker.
	ackHandler := newAWSACKHandler()
	client, err := in.pipeline.ConnectWith(beat.ClientConfig{
		EventListener: ackHandler.pipelineEventListener(),
		Processing: beat.ProcessingConfig{
			// This input only produces events with basic types so normalization
			// is not required.
			EventNormalization: boolPtr(false),
		},
	})
	if err != nil {
		in.workerWg.Done()
		return nil, fmt.Errorf("connecting to pipeline: %w", err)
	}
	return &sqsWorker{
		input:      in,
		client:     client,
		ackHandler: ackHandler,
		wg:         &in.workerWg,
	}, nil
}

func (w *sqsWorker) run(ctx, graceCtx context.Context) {
	defer func() {
		w.ackHandler.Close()
		w.client.Close()
		w.wg.Done()
	}()

	for graceCtx.Err() == nil {
		// Send a work request
		select {
		case <-graceCtx.Done():
			// Shutting down
			return
		case <-ctx.Done():
			// Requests will no longer be received
			// since the parent context has been
			// cancelled, so do not wait for this.
			// But do check to see whether we have
			// completed our pending publications.
			if w.pending.Load() == 0 {
				// If we have zero pending, we
				// can exit early.
				return
			}
		case w.input.workRequestChan <- struct{}{}:
		}
		// The request is sent, wait for a response
		select {
		case <-graceCtx.Done():
			return
		case msg := <-w.input.workResponseChan:
			w.processMessage(graceCtx, msg)
		case <-ctx.Done():
			// We're shutting down, so spin in the
			// loop until we have exceeded our
			// grace time, or we have no pending
			// messages.
		}
	}
}

func (w *sqsWorker) processMessage(ctx context.Context, msg types.Message) {
	publishCount := 0
	id := w.input.metrics.beginSQSWorker()
	result := w.input.msgHandler.ProcessSQS(ctx, &msg, func(e beat.Event) {
		w.client.Publish(e)
		publishCount++
	})

	if publishCount == 0 {
		// No events made it through (probably an error state), wrap up immediately
		result.Done()
	} else {
		w.pending.Add(1)
		// Add this result's Done callback to the pending ACKs list
		w.ackHandler.Add(publishCount, func() {
			result.Done()
			w.pending.Add(-1)
		})
	}

	w.input.metrics.endSQSWorker(id)
}

func (in *sqsReaderInput) startWorkers(ctx, graceCtx context.Context) {
	// Start the worker goroutines that will fetch messages via workRequestChan
	// and workResponseChan until the input shuts down.
	for i := 0; i < in.config.NumberOfWorkers; i++ {
		in.workerWg.Add(1)
		go func() {
			worker, err := in.newSQSWorker()
			if err != nil {
				in.log.Error(err)
				return
			}
			go worker.run(ctx, graceCtx)
		}()
	}
}

func (in *sqsReaderInput) logConfigSummary() {
	log := in.log
	log.Infof("AWS api_timeout is set to %v.", in.config.APITimeout)
	log.Infof("AWS region is set to %v.", in.awsConfig.Region)
	if in.awsConfig.Region != in.detectedRegion {
		log.Warnf("configured region disagrees with queue_url region (%q != %q): using %q", in.awsConfig.Region, in.detectedRegion, in.awsConfig.Region)
	}
	log.Infof("AWS SQS visibility_timeout is set to %v.", in.config.VisibilityTimeout)
	log.Infof("AWS SQS number_of_workers is set to %v.", in.config.NumberOfWorkers)

	if in.config.BackupConfig.GetBucketName() != "" {
		log.Warnf("You have the backup_to_bucket functionality activated with SQS. Please make sure to set appropriate destination buckets " +
			"or prefixes to avoid an infinite loop.")
	}
}

func (in *sqsReaderInput) createEventProcessor() (sqsProcessor, error) {
	fileSelectors := in.config.getFileSelectors()
	s3EventHandlerFactory := newS3ObjectProcessorFactory(in.metrics, in.s3, fileSelectors, in.config.BackupConfig)

	script, err := newScriptFromConfig(in.log.Named("sqs_script"), in.config.SQSScript)
	if err != nil {
		return nil, err
	}
	return newSQSS3EventProcessor(in.log.Named("sqs_s3_event"), in.metrics, in.sqs, script, in.config.VisibilityTimeout, in.config.SQSMaxReceiveCount, s3EventHandlerFactory), nil
}

// Read all pending requests and return their count. If block is true,
// waits until the result is at least 1, unless the context expires.
func channelRequestCount(
	ctx context.Context,
	requestChan chan struct{},
	block bool,
) int {
	requestCount := 0
	if block {
		// Wait until at least one request comes in.
		select {
		case <-ctx.Done():
			return 0
		case <-requestChan:
			requestCount++
		}
	}
	// Read as many requests as we can without blocking.
	for {
		select {
		case <-requestChan:
			requestCount++
		default:
			return requestCount
		}
	}
}
