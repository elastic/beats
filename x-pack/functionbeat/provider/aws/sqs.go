// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/goformation/cloudformation"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/core"
	"github.com/elastic/beats/x-pack/functionbeat/provider"
	"github.com/elastic/beats/x-pack/functionbeat/provider/aws/transformer"
)

const batchSize = 10

// SQSConfig is the configuration for the SQS event type.
type SQSConfig struct {
	Triggers     []*SQSTriggerConfig `config:"triggers"`
	Description  string              `config:"description"`
	Name         string              `config:"name" validate:"nonzero,required"`
	LambdaConfig *lambdaConfig       `config:",inline"`
}

// SQSTriggerConfig configuration for the current trigger.
type SQSTriggerConfig struct {
	EventSourceArn string `config:"event_source_arn"`
}

// Validate validates the configuration.
func (cfg *SQSConfig) Validate() error {
	if len(cfg.Triggers) == 0 {
		return errors.New("you need to specify at least one trigger")
	}
	return nil
}

// SQS receives events from the web service and forward them to elasticsearch.
type SQS struct {
	log    *logp.Logger
	config *SQSConfig
}

// NewSQS creates a new function to receives events from a SQS queue.
func NewSQS(provider provider.Provider, cfg *common.Config) (provider.Function, error) {
	config := &SQSConfig{LambdaConfig: DefaultLambdaConfig}
	if err := cfg.Unpack(config); err != nil {
		return nil, err
	}
	return &SQS{log: logp.NewLogger("sqs"), config: config}, nil
}

// Run starts the lambda function and wait for web triggers.
func (s *SQS) Run(_ context.Context, client core.Client) error {
	lambda.Start(s.createHandler(client))
	return nil
}

func (s *SQS) createHandler(client core.Client) func(request events.SQSEvent) error {
	return func(request events.SQSEvent) error {
		s.log.Debugf("The handler receives %d events", len(request.Records))

		events := transformer.SQS(request)
		if err := client.PublishAll(events); err != nil {
			s.log.Errorf("Could not publish events to the pipeline, error: %+v", err)
			return err
		}
		client.Wait()
		return nil
	}
}

// Name return the name of the lambda function.
func (s *SQS) Name() string {
	return "sqs"
}

// Template returns the cloudformation template for configuring the service with the specified triggers.
func (s *SQS) Template() *cloudformation.Template {
	template := cloudformation.NewTemplate()

	prefix := func(suffix string) string {
		return "fnb" + s.config.Name + suffix
	}

	for _, trigger := range s.config.Triggers {
		resourceName := prefix("SQS") + normalizeResourceName(trigger.EventSourceArn)
		template.Resources[resourceName] = &cloudformation.AWSLambdaEventSourceMapping{
			BatchSize:      batchSize,
			EventSourceArn: trigger.EventSourceArn,
			FunctionName:   cloudformation.GetAtt(prefix(""), "Arn"),
		}
	}
	return template
}

// LambdaConfig returns the configuration to use when creating the lambda.
func (s *SQS) LambdaConfig() *lambdaConfig {
	return s.config.LambdaConfig
}
