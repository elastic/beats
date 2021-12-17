// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"errors"
	"sort"

	"github.com/aws/aws-lambda-go/events"
	lambdarunner "github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/goformation/v4/cloudformation"
	"github.com/awslabs/goformation/v4/cloudformation/iam"
	"github.com/awslabs/goformation/v4/cloudformation/lambda"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/telemetry"
	"github.com/elastic/beats/v7/x-pack/functionbeat/provider/aws/aws/transformer"
)

const batchSize = 10

// SQSConfig is the configuration for the SQS event type.
type SQSConfig struct {
	Triggers     []*SQSTriggerConfig `config:"triggers"`
	Description  string              `config:"description"`
	Name         string              `config:"name" validate:"nonzero,required"`
	LambdaConfig *LambdaConfig       `config:",inline"`
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

// SQSDetails returns the details of the feature.
func SQSDetails() feature.Details {
	return feature.MakeDetails("SQS trigger", "receive events from a SQS queue", feature.Stable)
}

// Run starts the lambda function and wait for web triggers.
func (s *SQS) Run(_ context.Context, client pipeline.ISyncClient, t telemetry.T) error {
	t.AddTriggeredFunction()

	lambdarunner.Start(s.createHandler(client))
	return nil
}

func (s *SQS) createHandler(client pipeline.ISyncClient) func(request events.SQSEvent) error {
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
		return NormalizeResourceName("fnb" + s.config.Name + suffix)
	}

	for _, trigger := range s.config.Triggers {
		resourceName := prefix("SQS") + NormalizeResourceName(trigger.EventSourceArn)
		template.Resources[resourceName] = &lambda.EventSourceMapping{
			BatchSize:      batchSize,
			EventSourceArn: trigger.EventSourceArn,
			FunctionName:   cloudformation.GetAtt(prefix(""), "Arn"),
		}
	}
	return template
}

// Policies returns a slice of policies to add to the lambda role.
func (s *SQS) Policies() []iam.Role_Policy {
	resources := make([]string, len(s.config.Triggers))
	for idx, trigger := range s.config.Triggers {
		resources[idx] = trigger.EventSourceArn
	}

	// Give us a chance to generate the same document indenpendant of the changes,
	// to help with updates.
	sort.Strings(resources)

	// SQS Roles permissions:
	// - lambda:CreateEventSourceMapping
	// - lambda:ListEventSourceMappings
	// - lambda:ListFunctions
	//
	// Lambda Role permission
	// - sqs:ChangeMessageVisibility
	// - sqs:DeleteMessage
	// - sqs:GetQueueAttributes
	// - sqs:ReceiveMessage
	policies := []iam.Role_Policy{
		iam.Role_Policy{
			PolicyName: cloudformation.Join("-", []string{"fnb", "sqs", s.config.Name}),
			PolicyDocument: map[string]interface{}{
				"Statement": []map[string]interface{}{
					map[string]interface{}{
						"Action": []string{
							"sqs:ChangeMessageVisibility",
							"sqs:DeleteMessage",
							"sqs:GetQueueAttributes",
							"sqs:ReceiveMessage",
						},
						"Effect":   "Allow",
						"Resource": resources,
					},
				},
			},
		},
	}

	return policies
}

// LambdaConfig returns the configuration to use when creating the lambda.
func (s *SQS) LambdaConfig() *LambdaConfig {
	return s.config.LambdaConfig
}
