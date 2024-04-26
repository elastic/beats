// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go"

	"github.com/elastic/beats/v7/filebeat/beater"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	sqsRetryDelay                  = 10 * time.Second
	sqsApproximateNumberOfMessages = "ApproximateNumberOfMessages"
)

type sqsReaderInput struct {
	config    config
	awsConfig awssdk.Config
}

type sqsReader struct {
	maxMessagesInflight int
	activeMessages      atomic.Int
	sqs                 sqsAPI
	msgHandler          sqsProcessor
	log                 *logp.Logger
	metrics             *inputMetrics

	// The main loop sends incoming messages to workChan, and the worker
	// goroutines read from it.
	workChan chan types.Message

	// workerWg is used to wait on worker goroutines during shutdown
	workerWg sync.WaitGroup
}

func newSQSReaderInput(config config,
	awsConfig awssdk.Config,
	store beater.StateStore,
) (v2.Input, error) {
	return &sqsReaderInput{
		config:    config,
		awsConfig: awsConfig,
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
	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)
	configRegion := in.config.RegionName
	urlRegion, err := getRegionFromQueueURL(in.config.QueueURL, in.config.AWSConfig.Endpoint)
	if err != nil && configRegion == "" {
		// Only report an error if we don't have a configured region
		// to fall back on.
		return fmt.Errorf("failed to get AWS region from queue_url: %w", err)
	} else if configRegion != "" && configRegion != urlRegion {
		inputContext.Logger.Warnf("configured region disagrees with queue_url region (%q != %q): using %q", configRegion, urlRegion, urlRegion)
	}

	in.awsConfig.Region = urlRegion

	// Create SQS receiver and S3 notification processor.
	receiver, err := in.createSQSReceiver(inputContext, pipeline)
	if err != nil {
		return fmt.Errorf("failed to initialize sqs receiver: %w", err)
	}
	defer receiver.metrics.Close()

	// Poll metrics periodically in the background
	go pollSqsWaitingMetric(ctx, receiver)

	receiver.Receive(ctx)
	return nil
}

func (in *sqsReaderInput) createSQSReceiver(ctx v2.Context, pipeline beat.Pipeline) (*sqsReader, error) {
	sqsAPI := &awsSQSAPI{
		client: sqs.NewFromConfig(in.awsConfig, func(o *sqs.Options) {
			if in.config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
		}),
		queueURL:          in.config.QueueURL,
		apiTimeout:        in.config.APITimeout,
		visibilityTimeout: in.config.VisibilityTimeout,
		longPollWaitTime:  in.config.SQSWaitTime,
	}

	s3API := &awsS3API{
		client: s3.NewFromConfig(in.awsConfig, func(o *s3.Options) {
			if in.config.AWSConfig.FIPSEnabled {
				o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
			}
			o.UsePathStyle = in.config.PathStyle
		}),
	}

	log := ctx.Logger.With("queue_url", in.config.QueueURL)
	log.Infof("AWS api_timeout is set to %v.", in.config.APITimeout)
	log.Infof("AWS region is set to %v.", in.awsConfig.Region)
	log.Infof("AWS SQS visibility_timeout is set to %v.", in.config.VisibilityTimeout)
	log.Infof("AWS SQS max_number_of_messages is set to %v.", in.config.MaxNumberOfMessages)

	if in.config.BackupConfig.GetBucketName() != "" {
		log.Warnf("You have the backup_to_bucket functionality activated with SQS. Please make sure to set appropriate destination buckets" +
			"or prefixes to avoid an infinite loop.")
	}

	fileSelectors := in.config.FileSelectors
	if len(in.config.FileSelectors) == 0 {
		fileSelectors = []fileSelectorConfig{{ReaderConfig: in.config.ReaderConfig}}
	}
	script, err := newScriptFromConfig(log.Named("sqs_script"), in.config.SQSScript)
	if err != nil {
		return nil, err
	}
	metrics := newInputMetrics(ctx.ID, nil, in.config.MaxNumberOfMessages)

	s3EventHandlerFactory := newS3ObjectProcessorFactory(log.Named("s3"), metrics, s3API, fileSelectors, in.config.BackupConfig)

	sqsMessageHandler := newSQSS3EventProcessor(log.Named("sqs_s3_event"), metrics, sqsAPI, script, in.config.VisibilityTimeout, in.config.SQSMaxReceiveCount, pipeline, s3EventHandlerFactory)

	sqsReader := newSQSReader(log.Named("sqs"), metrics, sqsAPI, in.config.MaxNumberOfMessages, sqsMessageHandler)

	return sqsReader, nil
}

func newSQSReader(log *logp.Logger, metrics *inputMetrics, sqs sqsAPI, maxMessagesInflight int, msgHandler sqsProcessor) *sqsReader {
	if metrics == nil {
		// Metrics are optional. Initialize a stub.
		metrics = newInputMetrics("", nil, 0)
	}
	return &sqsReader{
		maxMessagesInflight: maxMessagesInflight,
		sqs:                 sqs,
		msgHandler:          msgHandler,
		log:                 log,
		metrics:             metrics,
		workChan:            make(chan types.Message),
	}
}

// The main loop of the reader, that fetches messages from SQS
// and forwards them to workers via workChan.
func (r *sqsReader) Receive(ctx context.Context) {
	r.startWorkers(ctx)
	r.readerLoop(ctx)

	// Close the work channel to signal to the workers that we're done,
	// then wait for them to finish.
	close(r.workChan)
	r.workerWg.Wait()
}

func (r *sqsReader) readerLoop(ctx context.Context) {
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

func (r *sqsReader) workerLoop(ctx context.Context) {
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

func (r *sqsReader) readMessages(ctx context.Context) []types.Message {
	// We try to read enough messages to bring activeMessages up to the
	// total worker count (plus one, to unblock us when workers are ready
	// for more messages)
	readCount := r.maxMessagesInflight + 1 - r.activeMessages.Load()
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

func (r *sqsReader) startWorkers(ctx context.Context) {
	// Start the worker goroutines that will process messages from workChan
	// until the input shuts down.
	for i := 0; i < r.maxMessagesInflight; i++ {
		r.workerWg.Add(1)
		go func() {
			defer r.workerWg.Done()
			r.workerLoop(ctx)
		}()
	}
}

func (r *sqsReader) GetApproximateMessageCount(ctx context.Context) (int, error) {
	attributes, err := r.sqs.GetQueueAttributes(ctx, []types.QueueAttributeName{sqsApproximateNumberOfMessages})
	if err == nil {
		if c, found := attributes[sqsApproximateNumberOfMessages]; found {
			if messagesCount, err := strconv.Atoi(c); err == nil {
				return messagesCount, nil
			}
		}
	}
	return -1, err
}

var errBadQueueURL = errors.New("QueueURL is not in format: https://sqs.{REGION_ENDPOINT}.{ENDPOINT}/{ACCOUNT_NUMBER}/{QUEUE_NAME} or https://{VPC_ENDPOINT}.sqs.{REGION_ENDPOINT}.vpce.{ENDPOINT}/{ACCOUNT_NUMBER}/{QUEUE_NAME}")

func getRegionFromQueueURL(queueURL, endpoint string) (string, error) {
	// get region from queueURL
	// Example for sqs queue: https://sqs.us-east-1.amazonaws.com/12345678912/test-s3-logs
	// Example for vpce: https://vpce-test.sqs.us-east-1.vpce.amazonaws.com/12345678912/sqs-queue
	u, err := url.Parse(queueURL)
	if err != nil {
		return "", fmt.Errorf(queueURL + " is not a valid URL")
	}
	if (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" {
		queueHostSplit := strings.SplitN(u.Host, ".", 3)
		// check for sqs queue url
		if len(queueHostSplit) == 3 && queueHostSplit[0] == "sqs" {
			if queueHostSplit[2] == endpoint || (endpoint == "" && strings.HasPrefix(queueHostSplit[2], "amazonaws.")) {
				return queueHostSplit[1], nil
			}
		}

		// check for vpce url
		queueHostSplitVPC := strings.SplitN(u.Host, ".", 5)
		if len(queueHostSplitVPC) == 5 && queueHostSplitVPC[1] == "sqs" {
			if queueHostSplitVPC[4] == endpoint || (endpoint == "" && strings.HasPrefix(queueHostSplitVPC[4], "amazonaws.")) {
				return queueHostSplitVPC[2], nil
			}
		}
	}
	return "", errBadQueueURL
}

func pollSqsWaitingMetric(ctx context.Context, receiver *sqsReader) {
	// Run GetApproximateMessageCount before start of timer to set initial count for sqs waiting metric
	// This is to avoid misleading values in metric when sqs messages are processed before the ticker channel kicks in
	if shouldReturn := updateMessageCount(receiver, ctx); shouldReturn {
		return
	}

	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if shouldReturn := updateMessageCount(receiver, ctx); shouldReturn {
				return
			}
		}
	}
}

// updateMessageCount runs GetApproximateMessageCount for the given context and updates the receiver metric with the count returning false on no error
// If there is an error, the metric is reinitialized to -1 and true is returned
func updateMessageCount(receiver *sqsReader, ctx context.Context) bool {
	count, err := receiver.GetApproximateMessageCount(ctx)

	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case sqsAccessDeniedErrorCode:
			// stop polling if auth error is encountered
			// Set it back to -1 because there is a permission error
			receiver.metrics.sqsMessagesWaiting.Set(int64(-1))
			return true
		}
	}

	receiver.metrics.sqsMessagesWaiting.Set(int64(count))
	return false
}
