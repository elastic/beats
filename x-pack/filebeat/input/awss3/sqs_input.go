// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"fmt"
	"sync"
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

	// The expected region based on the queue URL
	detectedRegion string

	// Workers send on workRequestChan to indicate they're ready for the next
	// message, and the reader loop replies on workResponseChan.
	workRequestChan  chan struct{}
	workResponseChan chan types.Message

	// workerWg is used to wait on worker goroutines during shutdown
	workerWg sync.WaitGroup
}

func newSQSReaderInput(config config, awsConfig awssdk.Config) (*sqsReaderInput, error) {
	detectedRegion := getRegionFromQueueURL(config.QueueURL, config.AWSConfig.Endpoint)
	if config.RegionName != "" {
		awsConfig.Region = config.RegionName
	} else if detectedRegion == "" {
		// Only report an error if we don't have a configured region
		// to fall back on.
		return nil, fmt.Errorf("failed to get AWS region from queue_url: %w", errBadQueueURL)
	}

	sqsAPI := &awsSQSAPI{
		client: sqs.NewFromConfig(awsConfig, func(o *sqs.Options) {
			if config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
		}),
		queueURL:          config.QueueURL,
		apiTimeout:        config.APITimeout,
		visibilityTimeout: config.VisibilityTimeout,
		longPollWaitTime:  config.SQSWaitTime,
	}

	s3API := &awsS3API{
		client: s3.NewFromConfig(awsConfig, func(o *s3.Options) {
			if config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
			o.UsePathStyle = config.PathStyle
		}),
	}

	return &sqsReaderInput{
		config:           config,
		awsConfig:        awsConfig,
		sqs:              sqsAPI,
		s3:               s3API,
		detectedRegion:   detectedRegion,
		workRequestChan:  make(chan struct{}, config.MaxNumberOfMessages),
		workResponseChan: make(chan types.Message),
	}, nil
}

func (in *sqsReaderInput) Name() string { return inputName }

func (in *sqsReaderInput) Test(ctx v2.TestContext) error {
	return nil
}

func (in *sqsReaderInput) Run(
	inputContext v2.Context,
	pipeline beat.Pipeline,
) error {
	in.log = inputContext.Logger.With("queue_url", in.config.QueueURL)
	in.logConfigSummary()

	in.metrics = newInputMetrics(inputContext.ID, nil, in.config.MaxNumberOfMessages)
	defer in.metrics.Close()

	var err error
	in.msgHandler, err = in.createEventProcessor(pipeline)
	if err != nil {
		return fmt.Errorf("failed to initialize sqs reader: %w", err)
	}

	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)

	// Poll metrics periodically in the background
	go messageCountMonitor{
		sqs:     in.sqs,
		metrics: in.metrics,
	}.run(ctx)

	// Start the main run loop
	in.run(ctx)

	return nil
}

func (in *sqsReaderInput) run(ctx context.Context) {
	in.startWorkers(ctx)
	in.readerLoop(ctx)

	in.workerWg.Wait()
}

func (in *sqsReaderInput) createEventProcessor(pipeline beat.Pipeline) (sqsProcessor, error) {
	fileSelectors := in.config.FileSelectors
	if len(in.config.FileSelectors) == 0 {
		fileSelectors = []fileSelectorConfig{{ReaderConfig: in.config.ReaderConfig}}
	}
	s3EventHandlerFactory := newS3ObjectProcessorFactory(in.log.Named("s3"), in.metrics, in.s3, fileSelectors, in.config.BackupConfig)

	script, err := newScriptFromConfig(in.log.Named("sqs_script"), in.config.SQSScript)
	if err != nil {
		return nil, err
	}
	return newSQSS3EventProcessor(in.log.Named("sqs_s3_event"), in.metrics, in.sqs, script, in.config.VisibilityTimeout, in.config.SQSMaxReceiveCount, pipeline, s3EventHandlerFactory), nil
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

func (in *sqsReaderInput) workerLoop(ctx context.Context) {
	for ctx.Err() == nil {
		// Send a work request
		select {
		case <-ctx.Done():
			// Shutting down
			return
		case in.workRequestChan <- struct{}{}:
		}
		// The request is sent, wait for a response
		select {
		case <-ctx.Done():
			return
		case msg := <-in.workResponseChan:
			start := time.Now()

			id := in.metrics.beginSQSWorker()
			if err := in.msgHandler.ProcessSQS(ctx, &msg); err != nil {
				in.log.Warnw("Failed processing SQS message.",
					"error", err,
					"message_id", *msg.MessageId,
					"elapsed_time_ns", time.Since(start))
			}
			in.metrics.endSQSWorker(id)
		}
	}
}

func (in *sqsReaderInput) startWorkers(ctx context.Context) {
	// Start the worker goroutines that will process messages from workChan
	// until the input shuts down.
	for i := 0; i < in.config.MaxNumberOfMessages; i++ {
		in.workerWg.Add(1)
		go func() {
			defer in.workerWg.Done()
			in.workerLoop(ctx)
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
	log.Infof("AWS SQS max_number_of_messages is set to %v.", in.config.MaxNumberOfMessages)

	if in.config.BackupConfig.GetBucketName() != "" {
		log.Warnf("You have the backup_to_bucket functionality activated with SQS. Please make sure to set appropriate destination buckets" +
			"or prefixes to avoid an infinite loop.")
	}
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
