// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"sort"

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
	log    *logp.Logger
	config *CloudwatchKinesisConfig
}

type CloudwatchKinesisConfig struct {
	*KinesisConfig `config:",inline"`
	Compressed     bool `config:"compressed"`
}

func defaultCloudwatchKinesisConfig() *CloudwatchKinesisConfig {
	return &CloudwatchKinesisConfig{
		&KinesisConfig{
			LambdaConfig: DefaultLambdaConfig,
		},
		false,
	}
}

// NewCloudwatchKinesis creates a new function to receives events from a kinesis stream.
func NewCloudwatchKinesis(provider provider.Provider, cfg *common.Config) (provider.Function, error) {
	config := defaultCloudwatchKinesisConfig()
	if err := cfg.Unpack(config); err != nil {
		return nil, err
	}
	return &CloudwatchKinesis{log: logp.NewLogger("cloudwatch_kinesis"), config: config}, nil
}

// CloudwatchKinesisDetails returns the details of the feature.
func CloudwatchKinesisDetails() *feature.Details {
	return feature.NewDetails("Cloudwatch logs via Kinesis trigger", "receive Cloudwatch logs from a Kinesis stream", feature.Experimental)
}

// Run starts the lambda function and wait for web triggers.
func (k *CloudwatchKinesis) Run(_ context.Context, client core.Client) error {
	lambda.Start(k.createHandler(client))
	return nil
}

func (k *CloudwatchKinesis) createHandler(client core.Client) func(request events.KinesisEvent) error {
	return func(request events.KinesisEvent) error {
		k.log.Debugf("The handler receives %d events", len(request.Records))

		events, err := transformer.CloudwatchKinesisEvent(request, k.config.Compressed)
		if err != nil {
			return err
		}

		if err := client.PublishAll(events); err != nil {
			k.log.Errorf("Could not publish events to the pipeline, error: %+v", err)
			return err
		}
		client.Wait()
		return nil
	}
}

// Name return the name of the lambda function.
func (k *CloudwatchKinesis) Name() string {
	return "cloudwatch_kinesis"
}

// LambdaConfig returns the configuration to use when creating the lambda.
func (k *CloudwatchKinesis) LambdaConfig() *LambdaConfig {
	return k.config.LambdaConfig
}

// Template returns the cloudformation template for configuring the service with the specified
// triggers.
func (k *CloudwatchKinesis) Template() *cloudformation.Template {
	template := cloudformation.NewTemplate()
	prefix := func(suffix string) string {
		return NormalizeResourceName("fnb" + k.config.Name + suffix)
	}

	for _, trigger := range k.config.Triggers {
		resourceName := prefix(k.Name() + trigger.EventSourceArn)
		template.Resources[resourceName] = &cloudformation.AWSLambdaEventSourceMapping{
			BatchSize:        trigger.BatchSize,
			EventSourceArn:   trigger.EventSourceArn,
			FunctionName:     cloudformation.GetAtt(prefix(""), "Arn"),
			StartingPosition: trigger.StartingPosition.String(),
		}
	}

	return template
}

// Policies returns a slice of policy to add to the lambda role.
func (k *CloudwatchKinesis) Policies() []cloudformation.AWSIAMRole_Policy {
	resources := make([]string, len(k.config.Triggers))
	for idx, trigger := range k.config.Triggers {
		resources[idx] = trigger.EventSourceArn
	}

	// Give us a chance to generate the same document indenpendant of the changes,
	// to help with updates.
	sort.Strings(resources)

	policies := []cloudformation.AWSIAMRole_Policy{
		cloudformation.AWSIAMRole_Policy{
			PolicyName: cloudformation.Join("-", []string{"fnb", "kinesis", k.config.Name}),
			PolicyDocument: map[string]interface{}{
				"Statement": []map[string]interface{}{
					map[string]interface{}{
						"Action": []string{
							"kinesis:GetRecords",
							"kinesis:GetShardIterator",
							"Kinesis:DescribeStream",
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
