// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/awslabs/goformation/cloudformation"
	merrors "github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/beatless/core"
	"github.com/elastic/beats/x-pack/beatless/provider"
	"github.com/elastic/beats/x-pack/beatless/provider/aws/transformer"
)

// CloudwatchLogsConfig is the configuration for the cloudwatchlogs event type.
type CloudwatchLogsConfig struct {
	Triggers    []*CloudwatchLogsTriggerConfig `config:"triggers"`
	Description string                         `config:"description"`
	Name        string                         `config:"name" validate:"nonzero,required"`
	Role        string                         `config:"role" validate:"nonzero,required"`
}

// CloudwatchLogsTriggerConfig is the configuration for the specific triggers for cloudwatch.
type CloudwatchLogsTriggerConfig struct {
	LogGroupName  string `config:"log_group_name" validate:"nonzero,required"`
	FilterName    string `config:"filter_name" validate:"nonzero,required"`
	FilterPattern string `config:"filter_pattern"`
}

// Validate validates the configuration.
func (cfg *CloudwatchLogsConfig) Validate() error {
	if len(cfg.Triggers) == 0 {
		return errors.New("you need to specify at least one trigger")
	}
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
	config := &CloudwatchLogsConfig{}
	if err := cfg.Unpack(config); err != nil {
		return nil, err
	}
	return &CloudwatchLogs{log: logp.NewLogger("cloudwatch_logs"), config: config}, nil
}

// Run start the AWS lambda handles and will transform any events received to the pipeline.
func (c *CloudwatchLogs) Run(_ context.Context, client core.Client) error {
	lambda.Start(c.createHandler(client))
	return nil
}

func (c *CloudwatchLogs) createHandler(client core.Client) func(request events.CloudwatchLogsEvent) error {
	return func(request events.CloudwatchLogsEvent) error {
		parsedEvent, err := request.AWSLogs.Parse()
		if err != nil {
			c.log.Errorf("could not parse events from cloudwatch logs, error: %s", err)
			return err
		}

		c.log.Debugf(
			"received %d events (logStream: %s, owner: %s, logGroup: %s, messageType: %s)",
			len(parsedEvent.LogEvents),
			parsedEvent.LogStream,
			parsedEvent.Owner,
			parsedEvent.LogGroup,
			parsedEvent.MessageType,
		)

		events := transformer.CloudwatchLogs(parsedEvent)
		if err := client.PublishAll(events); err != nil {
			c.log.Errorf("could not publish events to the pipeline, error: %s")
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

// AWSLambdaFunction add 'dependsOn' as a serializable parameters, for no good reason it's
// not supported.
type AWSLambdaFunction struct {
	*cloudformation.AWSLambdaFunction
	DependsOn []string
}

// Deploy executes a number of operation to make the state consistent for this specific lambda function.
func (c *CloudwatchLogs) Deploy(content []byte, awsCfg aws.Config) error {
	context := &executorContext{
		Content:     content,
		Name:        c.config.Name,
		Description: c.config.Description,
		Role:        c.config.Role,
		Runtime:     runtime,
		HandleName:  handlerName,
	}

	bucket := "mybucket-for-beatless"
	fnCodeKey := "beatless-deployment/beatless/ph/beatless.zip"

	template := cloudformation.NewTemplate()
	template.Resources["IAMRoleLambdaExecution"] = &cloudformation.AWSIAMRole{
		AssumeRolePolicyDocument: map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []interface{}{
				map[string]interface{}{
					"Action": "sts:AssumeRole",
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Service": []string{"lambda.amazonaws.com"},
					},
				},
			},
		},
		RoleName: "beatless-lambda",
	}

	template.Resources["btl"+c.config.Name] = &AWSLambdaFunction{
		AWSLambdaFunction: &cloudformation.AWSLambdaFunction{
			Code: &cloudformation.AWSLambdaFunction_Code{
				S3Bucket: bucket,
				S3Key:    fnCodeKey,
			},
			Description: "beatless " + c.config.Name + " lambda",
			Environment: &cloudformation.AWSLambdaFunction_Environment{
				Variables: map[string]string{
					"BEAT_STRICT_PERMS": "false",
					"ENABLED_FUNCTIONS": c.config.Name,
				},
			},
			FunctionName: c.config.Name,
			Role:         cloudformation.GetAtt("IAMRoleLambdaExecution", "Arn"),
			Runtime:      runtime,
			Handler:      handlerName,
		},
		DependsOn: []string{"IAMRoleLambdaExecution"},
	}

	j, err := template.JSON()
	if err != nil {
		return err
	}

	template.JSON()

	executer := newExecutor(c.log, context)
	executer.Add(newOpEnsureBucket(c.log, awsCfg, bucket))
	executer.Add(newOpUploadToBucket(c.log, awsCfg, bucket, fnCodeKey, content))
	executer.Add(newOpUploadToBucket(
		c.log,
		awsCfg,
		bucket,
		"beatless-deployment/beatless/ph/cloudformation-template-create.json",
		j,
	))
	executer.Add(newOpCreateCloudFormation(
		c.log,
		awsCfg,
		"https://s3.amazonaws.com/mybucket-for-beatless/beatless-deployment/beatless/ph/cloudformation-template-create.json",
		"stack-"+c.config.Name,
	))
	executer.Add(newOpWaitCloudFormation(
		c.log,
		awsCfg,
	))

	if err := executer.Execute(); err != nil {
		if rollbackErr := executer.Rollback(); rollbackErr != nil {
			return merrors.Wrapf(err, "could not rollback, error: %s", rollbackErr)
		}
		return err
	}
	return nil
}

// Update an existing lambda function.
func (c *CloudwatchLogs) Update(content []byte, awsCfg aws.Config) error {
	context := &executorContext{
		Content:     content,
		Name:        c.config.Name,
		Description: c.config.Description,
		Role:        c.config.Role,
		Runtime:     runtime,
		HandleName:  handlerName,
	}

	executer := newExecutor(c.log, context)

	executer.Add(newOpUpdateLambda(c.log, awsCfg))
	executer.Add(newOpUpdateAlias(c.log, awsCfg))

	for _, trigger := range c.config.Triggers {
		subscription := subscriptionFilter{
			LogGroupName:  trigger.LogGroupName,
			FilterName:    trigger.FilterName,
			FilterPattern: trigger.FilterPattern,
		}
		executer.Add(newOpAddSubscriptionFilter(c.log, awsCfg, subscription))
	}

	if err := executer.Execute(); err != nil {
		if rollbackErr := executer.Rollback(); rollbackErr != nil {
			return merrors.Wrapf(err, "could not rollback, error: %s", rollbackErr)
		}
		return err
	}
	return nil
}
