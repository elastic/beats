// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/pkg/errors"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/feature"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/go-concert/ctxtool"
)

const inputName = "s3"

func Plugin() v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Beta,
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

func (inp *s3Input) Name() string { return inputName }

func (inp *s3Input) Test(ctx v2.TestContext) error {
	_, err := awscommon.GetAWSCredentials(inp.config.AwsConfig)
	if err != nil {
		return errors.Wrap(err, "getAWSCredentials failed")
	}

	// XXX: more connection checks?
	return nil
}

func (inp *s3Input) Run(ctx v2.Context, pipeline beat.Pipeline) error {
	collector, err := inp.createCollector(ctx, pipeline)
	if err != nil {
		return err
	}

	defer collector.publisher.Close()
	collector.run()
	return ctx.Cancelation.Err()
}

func (inp *s3Input) createCollector(ctx v2.Context, pipeline beat.Pipeline) (*s3Collector, error) {
	// XXX: any fields we want to add to the logger
	log := ctx.Logger.With("queue_url", inp.config.QueueURL)

	client, err := pipeline.ConnectWith(beat.ClientConfig{
		CloseRef:   ctx.Cancelation,
		ACKHandler: newACKHandler(),
	})
	if err != nil {
		return nil, err
	}
	defer client.Close()

	regionName, err := getRegionFromQueueURL(inp.config.QueueURL)
	if err != nil {
		log.Errorf("failed to get region name from queueURL: %v", inp.config.QueueURL)
	} else {
		log = log.With("region", regionName)
	}

	awsConfig, err := awscommon.GetAWSCredentials(inp.config.AwsConfig)
	awsConfig.Region = regionName
	if err != nil {
		return nil, errors.Wrap(err, "getAWSCredentials failed")
	}

	visibilityTimeout := int64(inp.config.VisibilityTimeout.Seconds())
	log.Infof("visibility timeout is set to %v seconds", visibilityTimeout)
	log.Infof("aws api timeout is set to %v", inp.config.APITimeout)

	svcSQS := sqs.New(awscommon.EnrichAWSConfigWithEndpoint(inp.config.AwsConfig.Endpoint, "sqs", regionName, awsConfig))
	svcS3 := s3.New(awscommon.EnrichAWSConfigWithEndpoint(inp.config.AwsConfig.Endpoint, "s3", regionName, awsConfig))

	log.Infof("S3 input for '%v' is started.", inp.config.QueueURL)
	defer log.Infof("S3 input for '%v' is started.", inp.config.QueueURL)

	return &s3Collector{
		cancelation:       ctxtool.FromCanceller(ctx.Cancelation),
		logger:            log,
		config:            &inp.config,
		publisher:         client,
		visibilityTimeout: visibilityTimeout,
		sqs:               svcSQS,
		s3:                svcS3,
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
