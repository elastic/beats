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
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/core"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/telemetry"
	"github.com/elastic/beats/v7/x-pack/functionbeat/provider/aws/aws/transformer"
)

// SQSConfig is the configuration for the S3 event type.
type S3Config struct {
	Triggers     []*S3TriggerConfig `config:"triggers"`
	Description  string             `config:"description"`
	Name         string             `config:"name" validate:"nonzero,required"`
	LambdaConfig *LambdaConfig      `config:",inline"`
}

// S3TriggerConfig configuration for the current trigger.
type S3TriggerConfig struct {
	EventSourceArn string `config:"event_source_arn"`
}

// Validate validates the configuration.
func (cfg *S3Config) Validate() error {
	if len(cfg.Triggers) == 0 {
		return errors.New("you need to specify at least one trigger")
	}
	return nil
}

// S3 receives events from the web service and forward them to elasticsearch.
type S3 struct {
	log    *logp.Logger
	config *S3Config
}

// NewS3 creates a new function to receives events from a S3 queue.
func NewS3(provider provider.Provider, cfg *common.Config) (provider.Function, error) {
	config := &S3Config{LambdaConfig: DefaultLambdaConfig}
	if err := cfg.Unpack(config); err != nil {
		return nil, err
	}
	return &S3{log: logp.NewLogger("S3"), config: config}, nil
}

// SQSDetails returns the details of the feature.
func S3Details() feature.Details {
	return feature.MakeDetails("S3 trigger", "receive events from S3 notifications", feature.Stable)
}

// Run starts the lambda function and wait for web triggers.
func (s *S3) Run(_ context.Context, client core.Client, t telemetry.T) error {
	t.AddTriggeredFunction()

	lambdarunner.Start(s.createHandler(client))
	return nil
}

func (s *S3) createHandler(client core.Client) func(request events.S3Event) error {
	return func(request events.S3Event) error {
		s.log.Debugf("The handler receives %d events", len(request.Records))
		events, err := transformer.S3GetEvents(request)

		if err != nil {
			s.log.Errorf("Error retrieving object from bucket, error: %+v", err)
			return err
		}

		if err := client.PublishAll(events); err != nil {
			s.log.Errorf("Could not publish events to the pipeline, error: %+v", err)
			return err
		}
		client.Wait()
		return nil
	}
}

// Name return the name of the lambda function.
func (s *S3) Name() string {
	return "s3"
}

// Template returns the cloudformation template for configuring the service with the specified triggers.
func (s *S3) Template() *cloudformation.Template {
	template := cloudformation.NewTemplate()

	prefix := func(suffix string) string {
		return NormalizeResourceName("fnb" + s.config.Name + suffix)
	}

	for _, trigger := range s.config.Triggers {
		resourceName := prefix("s3") + NormalizeResourceName(trigger.EventSourceArn)
		template.Resources[resourceName] = &lambda.EventSourceMapping{
			BatchSize:      batchSize,
			EventSourceArn: trigger.EventSourceArn,
			FunctionName:   cloudformation.GetAtt(prefix(""), "Arn"),
		}
	}
	return template
}

// Policies returns a slice of policies to add to the lambda role.
func (s *S3) Policies() []iam.Role_Policy {
	resources := make([]string, len(s.config.Triggers))
	for idx, trigger := range s.config.Triggers {
		resources[idx] = trigger.EventSourceArn
	}

	// Give us a chance to generate the same document indenpendant of the changes,
	// to help with updates.
	sort.Strings(resources)

	// S3 Roles permissions:
	// - lambda:invokeFunction
	//
	// Lambda Role permission
	// - s3:GetObject
	// - kms:GenerateDataKey ##if KMS encryption enabled
	// - kms:Decrypt ##if KMS encryption enabled
	policies := []iam.Role_Policy{
		iam.Role_Policy{
			PolicyName: cloudformation.Join("-", []string{"fnb", "s3", s.config.Name}),
			PolicyDocument: map[string]interface{}{
				"Statement": []map[string]interface{}{
					map[string]interface{}{
						"Action": []string{
							"s3:GetObject",
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
func (s *S3) LambdaConfig() *LambdaConfig {
	return s.config.LambdaConfig
}
