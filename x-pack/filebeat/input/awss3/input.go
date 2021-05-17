// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/go-concert/ctxtool"
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

// s3Input is a input for s3
type s3Input struct {
	config config
}

func newInput(config config) (*s3Input, error) {
	return &s3Input{config: config}, nil
}

func (in *s3Input) Name() string { return inputName }

func (in *s3Input) Test(ctx v2.TestContext) error {
	_, err := awscommon.GetAWSCredentials(in.config.AwsConfig)
	if err != nil {
		return fmt.Errorf("getAWSCredentials failed: %w", err)
	}
	return nil
}

func (in *s3Input) Run(ctx v2.Context, pipeline beat.Pipeline) error {
	collector, err := in.createCollector(ctx, pipeline)
	if err != nil {
		return err
	}

	defer collector.metrics.Close()
	defer collector.publisher.Close()
	collector.run()

	if ctx.Cancelation.Err() == context.Canceled {
		return nil
	} else {
		return ctx.Cancelation.Err()
	}
}

func (in *s3Input) createCollector(ctx v2.Context, pipeline beat.Pipeline) (*s3Collector, error) {
	log := ctx.Logger.With("queue_url", in.config.QueueURL)

	client, err := pipeline.ConnectWith(beat.ClientConfig{
		CloseRef:   ctx.Cancelation,
		ACKHandler: newACKHandler(),
	})
	if err != nil {
		return nil, err
	}

	regionName, err := getRegionFromQueueURL(in.config.QueueURL, in.config.AwsConfig.Endpoint)
	if err != nil {
		err := fmt.Errorf("getRegionFromQueueURL failed: %w", err)
		log.Error(err)
		return nil, err
	} else {
		log = log.With("region", regionName)
	}

	awsConfig, err := awscommon.GetAWSCredentials(in.config.AwsConfig)
	if err != nil {
		return nil, fmt.Errorf("getAWSCredentials failed: %w", err)
	}
	awsConfig.Region = regionName

	visibilityTimeout := int64(in.config.VisibilityTimeout.Seconds())
	log.Infof("visibility timeout is set to %v seconds", visibilityTimeout)
	log.Infof("aws api timeout is set to %v", in.config.APITimeout)

	s3Servicename := "s3"
	if in.config.FipsEnabled {
		s3Servicename = "s3-fips"
	}

	log.Debug("s3 service name = ", s3Servicename)
	log.Debug("s3 input config max_number_of_messages = ", in.config.MaxNumberOfMessages)
	log.Debug("s3 input config endpoint = ", in.config.AwsConfig.Endpoint)
	metricRegistry := monitoring.GetNamespace("dataset").GetRegistry()
	return &s3Collector{
		cancellation:      ctxtool.FromCanceller(ctx.Cancelation),
		logger:            log,
		config:            &in.config,
		publisher:         client,
		visibilityTimeout: visibilityTimeout,
		sqs:               sqs.New(awscommon.EnrichAWSConfigWithEndpoint(in.config.AwsConfig.Endpoint, "sqs", regionName, awsConfig)),
		s3:                s3.New(awscommon.EnrichAWSConfigWithEndpoint(in.config.AwsConfig.Endpoint, s3Servicename, regionName, awsConfig)),
		metrics:           newInputMetrics(metricRegistry, ctx.ID),
	}, nil
}

func newACKHandler() beat.ACKer {
	return acker.ConnectionOnly(
		acker.EventPrivateReporter(func(_ int, privates []interface{}) {
			for _, private := range privates {
				if s3Context, ok := private.(*s3Context); ok {
					s3Context.done()
				}
			}
		}),
	)
}
