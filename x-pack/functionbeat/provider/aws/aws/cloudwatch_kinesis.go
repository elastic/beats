// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/goformation/cloudformation"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/function/core"
	"github.com/elastic/beats/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/x-pack/functionbeat/provider/aws/aws/transformer"
)

// CloudwatchKinesis receives events from a kinesis stream and forward them to elasticsearch.
type CloudwatchKinesis struct {
	*Kinesis
	log    *logp.Logger
	config *CloudwatchKinesisConfig
}

// CloudwatchKinesisConfig stores the configuration of Kinesis and additional options on decompressing data.
type CloudwatchKinesisConfig struct {
	*KinesisConfig `config:",inline"`
	Base64Encoded  bool `config:"base64_encoded"`
	Compressed     bool `config:"compressed"`
}

// NewCloudwatchKinesis creates a new function to receives events from a kinesis stream.
func NewCloudwatchKinesis(provider provider.Provider, cfg *common.Config) (provider.Function, error) {
	config := defaultCloudwatchKinesisConfig()
	if err := cfg.Unpack(config); err != nil {
		return nil, err
	}

	logger := logp.NewLogger("cloudwatch_logs_kinesis")

	return &CloudwatchKinesis{
		Kinesis: &Kinesis{
			config: config.KinesisConfig,
			log:    logger,
		},
		log:    logger,
		config: config,
	}, nil
}

func defaultCloudwatchKinesisConfig() *CloudwatchKinesisConfig {
	return &CloudwatchKinesisConfig{
		&KinesisConfig{
			LambdaConfig: DefaultLambdaConfig,
		},
		false,
		true,
	}
}

// CloudwatchKinesisDetails returns the details of the feature.
func CloudwatchKinesisDetails() *feature.Details {
	return feature.NewDetails("Cloudwatch logs via Kinesis trigger", "receive Cloudwatch logs from a Kinesis stream", feature.Experimental)
}

// Run starts the lambda function and wait for web triggers.
func (c *CloudwatchKinesis) Run(_ context.Context, client core.Client) error {
	lambda.Start(c.createHandler(client))
	return nil
}

func (c *CloudwatchKinesis) createHandler(client core.Client) func(request events.KinesisEvent) error {
	return func(request events.KinesisEvent) error {
		c.log.Debugf("The handler receives %d events", len(request.Records))

		events, err := transformer.CloudwatchKinesisEvent(request, c.config.Base64Encoded, c.config.Compressed)
		if err != nil {
			return err
		}

		if err := client.PublishAll(events); err != nil {
			c.log.Errorf("Could not publish events to the pipeline, error: %+v", err)
			return err
		}
		client.Wait()
		return nil
	}
}

// Name return the name of the lambda function.
func (c *CloudwatchKinesis) Name() string {
	return "cloudwatch_logs_kinesis"
}

// LambdaConfig returns the configuration to use when creating the lambda.
func (c *CloudwatchKinesis) LambdaConfig() *LambdaConfig {
	return c.config.LambdaConfig
}

// Template returns the cloudformation template for configuring the service with the specified
// triggers.
func (c *CloudwatchKinesis) Template() *cloudformation.Template {
	return c.Kinesis.Template()
}

// Policies returns a slice of policy to add to the lambda role.
func (c *CloudwatchKinesis) Policies() []cloudformation.AWSIAMRole_Policy {
	return c.Kinesis.Policies()
}
