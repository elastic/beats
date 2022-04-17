// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	lambdarunner "github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/goformation/v4/cloudformation"
	"github.com/awslabs/goformation/v4/cloudformation/iam"
	"github.com/awslabs/goformation/v4/cloudformation/lambda"
	"github.com/awslabs/goformation/v4/cloudformation/policies"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/feature"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/publisher/pipeline"
	"github.com/menderesk/beats/v7/x-pack/functionbeat/function/provider"
	"github.com/menderesk/beats/v7/x-pack/functionbeat/function/telemetry"
	"github.com/menderesk/beats/v7/x-pack/functionbeat/provider/aws/aws/transformer"
)

var (
	logGroupNamePattern = "^[\\.\\-_/#A-Za-z0-9]+$"
	logGroupNameRE      = regexp.MustCompile(logGroupNamePattern)
)

// CloudwatchLogsConfig is the configuration for the cloudwatchlogs event type.
type CloudwatchLogsConfig struct {
	Triggers     []*CloudwatchLogsTriggerConfig `config:"triggers"`
	Description  string                         `config:"description"`
	Name         string                         `config:"name" validate:"nonzero,required"`
	LambdaConfig *LambdaConfig                  `config:",inline"`
}

// CloudwatchLogsTriggerConfig is the configuration for the specific triggers for cloudwatch.
type CloudwatchLogsTriggerConfig struct {
	LogGroupName  logGroupName `config:"log_group_name" validate:"nonzero,required"`
	FilterPattern string       `config:"filter_pattern"`
}

// Validate validates the configuration.
func (cfg *CloudwatchLogsConfig) Validate() error {
	if len(cfg.Triggers) == 0 {
		return errors.New("you need to specify at least one trigger")
	}
	return nil
}

// DOC: see validations rules at https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_CreateLogGroup.html
type logGroupName string

// Unpack takes a string and validate the log group format.
func (l *logGroupName) Unpack(s string) error {
	const max = 512
	const min = 1

	if len(s) > max {
		return fmt.Errorf("log group name '%s' is too long, maximum length is %d", s, max)
	}

	if len(s) < min {
		return fmt.Errorf("log group name too short, minimum length is %d", min)
	}

	if !logGroupNameRE.MatchString(s) {
		return fmt.Errorf(
			"invalid characters in log group name '%s', name must match regular expression: '%s'",
			s,
			logGroupNamePattern,
		)
	}
	*l = logGroupName(s)
	return nil
}

// CloudwatchLogs receives CloudwatchLogs events from a lambda function and forward the logs to
// an Elasticsearch cluster.
type CloudwatchLogs struct {
	log    *logp.Logger
	config *CloudwatchLogsConfig
}

// NewCloudwatchLogs create a new function to listen to cloudwatch logs events.
func NewCloudwatchLogs(provider provider.Provider, cfg *common.Config) (provider.Function, error) {
	config := &CloudwatchLogsConfig{
		LambdaConfig: DefaultLambdaConfig,
	}
	if err := cfg.Unpack(config); err != nil {
		return nil, err
	}
	return &CloudwatchLogs{log: logp.NewLogger("cloudwatch_logs"), config: config}, nil
}

// CloudwatchLogsDetails returns the details of the feature.
func CloudwatchLogsDetails() feature.Details {
	return feature.MakeDetails("Cloudwatch Logs trigger", "receive events from cloudwatch logs.", feature.Stable)
}

// Run start the AWS lambda handles and will transform any events received to the pipeline.
func (c *CloudwatchLogs) Run(_ context.Context, client pipeline.ISyncClient, t telemetry.T) error {
	t.AddTriggeredFunction()

	lambdarunner.Start(c.createHandler(client))
	return nil
}

func (c *CloudwatchLogs) createHandler(
	client pipeline.ISyncClient,
) func(request events.CloudwatchLogsEvent) error {
	return func(request events.CloudwatchLogsEvent) error {
		parsedEvent, err := request.AWSLogs.Parse()
		if err != nil {
			c.log.Errorf("Could not parse events from cloudwatch logs, error: %+v", err)
			return err
		}

		c.log.Debugf(
			"The handler receives %d events (logStream: %s, owner: %s, logGroup: %s, messageType: %s)",
			len(parsedEvent.LogEvents),
			parsedEvent.LogStream,
			parsedEvent.Owner,
			parsedEvent.LogGroup,
			parsedEvent.MessageType,
		)

		events := transformer.CloudwatchLogs(parsedEvent)

		if err := client.PublishAll(events); err != nil {
			c.log.Errorf("Could not publish events to the pipeline, error: %+v", err)
			return err
		}
		client.Wait()
		return nil
	}
}

// Name returns the name of the function.
func (c CloudwatchLogs) Name() string {
	return "cloudwatch_logs"
}

// AWSLogsSubscriptionFilter overrides the type from goformation to allow to pass an empty string.
// The API support an empty string, but requires one, the original type does not permit that.
type AWSLogsSubscriptionFilter struct {
	DestinationArn string `json:"DestinationArn,omitempty"`
	FilterPattern  string `json:"FilterPattern"`
	LogGroupName   string `json:"LogGroupName,omitempty"`
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSLogsSubscriptionFilter) MarshalJSON() ([]byte, error) {
	type Properties AWSLogsSubscriptionFilter
	return json.Marshal(&struct {
		Type           string
		Properties     Properties
		DeletionPolicy policies.DeletionPolicy `json:"DeletionPolicy,omitempty"`
	}{
		Type:       r.AWSCloudFormationType(),
		Properties: (Properties)(r),
	})
}

// AWSCloudFormationType return the AWS type.
func (r *AWSLogsSubscriptionFilter) AWSCloudFormationType() string {
	return "AWS::Logs::SubscriptionFilter"
}

// Template returns the cloudformation template for configuring the service with the specified triggers.
func (c *CloudwatchLogs) Template() *cloudformation.Template {
	prefix := func(suffix string) string {
		return NormalizeResourceName("fnb" + c.config.Name + suffix)
	}

	template := cloudformation.NewTemplate()
	for idx, trigger := range c.config.Triggers {
		// doc: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-permission.html
		template.Resources[prefix("Permission"+strconv.Itoa(idx))] = &lambda.Permission{
			Action:       "lambda:InvokeFunction",
			FunctionName: cloudformation.GetAtt(prefix(""), "Arn"),
			Principal: cloudformation.Join("", []string{
				"logs.",
				cloudformation.Ref("AWS::Region"), // Use the configuration region.
				".",
				cloudformation.Ref("AWS::URLSuffix"), // awsamazon.com or .com.ch
			}),
			SourceArn: cloudformation.Join(
				"",
				[]string{
					"arn:",
					cloudformation.Ref("AWS::Partition"),
					":logs:",
					cloudformation.Ref("AWS::Region"),
					":",
					cloudformation.Ref("AWS::AccountId"),
					":log-group:",
					string(trigger.LogGroupName),
					":*",
				},
			),
		}

		// doc: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-logs-subscriptionfilter.html
		template.Resources[prefix("SF")+NormalizeResourceName(string(trigger.LogGroupName))] = &AWSLogsSubscriptionFilter{
			DestinationArn: cloudformation.GetAtt(prefix(""), "Arn"),
			FilterPattern:  trigger.FilterPattern,
			LogGroupName:   string(trigger.LogGroupName),
		}
	}
	return template
}

// LambdaConfig returns the configuration to use when creating the lambda.
func (c *CloudwatchLogs) LambdaConfig() *LambdaConfig {
	return c.config.LambdaConfig
}

// Policies returns a slice of policy to add to the lambda.
func (c *CloudwatchLogs) Policies() []iam.Role_Policy {
	return []iam.Role_Policy{}
}
