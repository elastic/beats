// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"sync"

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

func newSQSReaderInput(config config, awsConfig awssdk.Config) (v2.Input, error) {
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
