// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

const inputName = "aws-s3"

func Plugin() v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Collect logs from s3",
		Manager:    v2.ConfigureWith(configure),
	}
}

func configure(cfg *common.Config) (v2.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newInput(config)
}

// s3Input is a input for reading logs from S3 when triggered by an SQS message.
type s3Input struct {
	config    config
	awsConfig awssdk.Config
}

func newInput(config config) (*s3Input, error) {
	awsConfig, err := awscommon.InitializeAWSConfig(config.AWSConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS credentials: %w", err)
	}

	regionName, err := getRegionFromQueueURL(config.QueueURL, config.AWSConfig.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS region from queue_url: %w", err)
	}
	awsConfig.Region = regionName

	return &s3Input{
		config:    config,
		awsConfig: awsConfig,
	}, nil
}

func (in *s3Input) Name() string { return inputName }

func (in *s3Input) Test(ctx v2.TestContext) error {
	return nil
}

func (in *s3Input) Run(inputContext v2.Context, pipeline beat.Pipeline) error {
	// Wrap input Context's cancellation Done channel a context.Context. This
	// goroutine stops with the parent closes the Done channel.
	ctx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputContext.Cancelation.Done():
		case <-ctx.Done():
		}
	}()
	defer cancelInputCtx()

	// Create client for publishing events and receive notification of their ACKs.
	client, err := pipeline.ConnectWith(beat.ClientConfig{
		CloseRef:   inputContext.Cancelation,
		ACKHandler: newEventACKHandler(),
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline client: %w", err)
	}
	defer client.Close()

	// Create SQS receiver and S3 notification processor.
	receiver, err := in.createSQSReceiver(inputContext, client)
	if err != nil {
		return fmt.Errorf("failed to initialize sqs receiver: %w", err)
	}
	defer receiver.metrics.Close()

	if err := receiver.Receive(ctx); err != nil {
		return err
	}

	return nil
}

func (in *s3Input) createSQSReceiver(ctx v2.Context, client beat.Client) (*sqsReader, error) {
	s3ServiceName := "s3"
	if in.config.FIPSEnabled {
		s3ServiceName = "s3-fips"
	}

	sqsAPI := &awsSQSAPI{
		client:            sqs.New(awscommon.EnrichAWSConfigWithEndpoint(in.config.AWSConfig.Endpoint, "sqs", in.awsConfig.Region, in.awsConfig)),
		queueURL:          in.config.QueueURL,
		apiTimeout:        in.config.APITimeout,
		visibilityTimeout: in.config.VisibilityTimeout,
		longPollWaitTime:  in.config.SQSWaitTime,
	}

	s3API := &awsS3API{
		client: s3.New(awscommon.EnrichAWSConfigWithEndpoint(in.config.AWSConfig.Endpoint, s3ServiceName, in.awsConfig.Region, in.awsConfig)),
	}

	log := ctx.Logger.With("queue_url", in.config.QueueURL)
	log.Infof("AWS api_timeout is set to %v.", in.config.APITimeout)
	log.Infof("AWS region is set to %v.", in.awsConfig.Region)
	log.Infof("AWS SQS visibility_timeout is set to %v.", in.config.VisibilityTimeout)
	log.Infof("AWS SQS max_number_of_messages is set to %v.", in.config.MaxNumberOfMessages)
	log.Debugf("AWS S3 service name is %v.", s3ServiceName)

	metricRegistry := monitoring.GetNamespace("dataset").GetRegistry()
	metrics := newInputMetrics(metricRegistry, ctx.ID)

	fileSelectors := in.config.FileSelectors
	if len(in.config.FileSelectors) == 0 {
		fileSelectors = []fileSelectorConfig{{ReaderConfig: in.config.ReaderConfig}}
	}
	s3EventHandlerFactory := newS3ObjectProcessorFactory(log.Named("s3"), metrics, s3API, client, fileSelectors)
	sqsMessageHandler := newSQSS3EventProcessor(log.Named("sqs_s3_event"), metrics, sqsAPI, in.config.VisibilityTimeout, in.config.SQSMaxReceiveCount, s3EventHandlerFactory)
	sqsReader := newSQSReader(log.Named("sqs"), metrics, sqsAPI, in.config.MaxNumberOfMessages, sqsMessageHandler)

	return sqsReader, nil
}

func getRegionFromQueueURL(queueURL string, endpoint string) (string, error) {
	// get region from queueURL
	// Example: https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs
	url, err := url.Parse(queueURL)
	if err != nil {
		return "", fmt.Errorf(queueURL + " is not a valid URL")
	}
	if url.Scheme == "https" && url.Host != "" {
		queueHostSplit := strings.Split(url.Host, ".")
		if len(queueHostSplit) > 2 && (strings.Join(queueHostSplit[2:], ".") == endpoint || (endpoint == "" && queueHostSplit[2] == "amazonaws")) {
			return queueHostSplit[1], nil
		}
	}
	return "", fmt.Errorf("QueueURL is not in format: https://sqs.{REGION_ENDPOINT}.{ENDPOINT}/{ACCOUNT_NUMBER}/{QUEUE_NAME}")
}
