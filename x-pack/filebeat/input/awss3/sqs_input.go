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
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/elastic-agent-libs/logp"
)

type sqsReaderInput struct {
	config              config
	awsConfig           awssdk.Config
	maxMessagesInFlight int
	activeMessages      atomic.Int
	sqs                 sqsAPI
	s3                  s3API
	msgHandler          sqsProcessor
	log                 *logp.Logger
	metrics             *inputMetrics

	// The expected region based on the queue URL
	detectedRegion string

	// The main loop sends incoming messages to workChan, and the worker
	// goroutines read from it.
	workChan chan types.Message

	// workerWg is used to wait on worker goroutines during shutdown
	workerWg sync.WaitGroup
}

func newSQSReaderInput(config config, awsConfig awssdk.Config) (v2.Input, error) {
	detectedRegion, err := getRegionFromQueueURL(config.QueueURL, config.AWSConfig.Endpoint)
	if config.RegionName != "" {
		awsConfig.Region = config.RegionName
	} else if err != nil {
		// Only report an error if we don't have a configured region
		// to fall back on.
		return nil, fmt.Errorf("failed to get AWS region from queue_url: %w", err)
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
		config:         config,
		awsConfig:      awsConfig,
		sqs:            sqsAPI,
		s3:             s3API,
		detectedRegion: detectedRegion,
		workChan:       make(chan types.Message),
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
	// Create SQS reader and S3 notification processor.
	err := in.initialize(inputContext, pipeline)
	if err != nil {
		return fmt.Errorf("failed to initialize sqs receiver: %w", err)
	}
	defer in.metrics.Close()

	// Poll metrics periodically in the background
	go pollSqsWaitingMetric(inputContext.Cancelation, in.sqs, in.metrics)

	in.Receive(inputContext.Cancelation)
	return nil
}

func (in *sqsReaderInput) initialize(ctx v2.Context, pipeline beat.Pipeline) error {
	log := ctx.Logger.With("queue_url", in.config.QueueURL)
	in.log = log
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

	in.metrics = newInputMetrics(ctx.ID, nil, in.config.MaxNumberOfMessages)

	var err error
	in.msgHandler, err = in.createEventProcessor(pipeline)
	if err != nil {
		return err
	}

	in.maxMessagesInFlight = in.config.MaxNumberOfMessages
	return nil
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

// The main loop of the reader, that fetches messages from SQS
// and forwards them to workers via workChan.
func (r *sqsReaderInput) Receive(canceler v2.Canceler) {
	ctx := v2.GoContextFromCanceler(canceler)
	r.startWorkers(ctx)
	r.readerLoop(ctx)

	// Close the work channel to signal to the workers that we're done,
	// then wait for them to finish.
	close(r.workChan)
	r.workerWg.Wait()
}

func (r *sqsReaderInput) readerLoop(ctx context.Context) {
	for ctx.Err() == nil {
		msgs := r.readMessages(ctx)

		for _, msg := range msgs {
			select {
			case <-ctx.Done():
			case r.workChan <- msg:
			}
		}
	}
}

func (r *sqsReaderInput) workerLoop(ctx context.Context) {
	for msg := range r.workChan {
		start := time.Now()

		id := r.metrics.beginSQSWorker()
		if err := r.msgHandler.ProcessSQS(ctx, &msg); err != nil {
			r.log.Warnw("Failed processing SQS message.",
				"error", err,
				"message_id", *msg.MessageId,
				"elapsed_time_ns", time.Since(start))
		}
		r.metrics.endSQSWorker(id)
		r.activeMessages.Dec()
	}
}

func (r *sqsReaderInput) readMessages(ctx context.Context) []types.Message {
	// We try to read enough messages to bring activeMessages up to the
	// total worker count (plus one, to unblock us when workers are ready
	// for more messages)
	readCount := r.config.MaxNumberOfMessages + 1 - r.activeMessages.Load()
	if readCount <= 0 {
		return nil
	}
	msgs, err := r.sqs.ReceiveMessage(ctx, readCount)
	for err != nil && ctx.Err() == nil {
		r.log.Warnw("SQS ReceiveMessage returned an error. Will retry after a short delay.", "error", err)
		// Wait for the retry delay, but stop early if the context is cancelled.
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(sqsRetryDelay):
		}
		msgs, err = r.sqs.ReceiveMessage(ctx, readCount)
	}
	r.activeMessages.Add(len(msgs))
	r.log.Debugf("Received %v SQS messages.", len(msgs))
	r.metrics.sqsMessagesReceivedTotal.Add(uint64(len(msgs)))
	return msgs
}

func (r *sqsReaderInput) startWorkers(ctx context.Context) {
	// Start the worker goroutines that will process messages from workChan
	// until the input shuts down.
	for i := 0; i < r.config.MaxNumberOfMessages; i++ {
		r.workerWg.Add(1)
		go func() {
			defer r.workerWg.Done()
			r.workerLoop(ctx)
		}()
	}
}
