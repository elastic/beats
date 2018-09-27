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
	merrors "github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/beatless/core"
	"github.com/elastic/beats/x-pack/beatless/provider"
	"github.com/elastic/beats/x-pack/beatless/provider/aws/transformer"
)

// CloudwatchLogsConfig is the configuration for the cloudwatchlogs event type.
type CloudwatchLogsConfig struct {
	Triggers    []*CloudwatchLogsTriggerConfig
	Description string `config:"description"`
	Name        string `config:"name" validate:"nonzero,required"`
	Role        string `config:"role" validate:"nonzero,required"`
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
	lambda.Start(func(request events.CloudwatchLogsEvent) error {
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
	})
	return nil
}

// Name returns the name of the function.
func (c CloudwatchLogs) Name() string {
	return "cloudwatch_logs"
}

// Deploy returns the list of operation that we need to execute on AWS lambda after installing the
// function.
func (c *CloudwatchLogs) Deploy(content []byte, awsCfg aws.Config) error {
	context := &executerContext{
		Content:     content,
		Name:        c.config.Name,
		Description: c.config.Description,
		Role:        c.config.Role,
		Runtime:     runtime,
		HandleName:  handlerName,
	}

	executer := newExecuter(c.log, context)
	executer.Add(newOpCreateLambda(c.log, awsCfg))
	executer.Add(newOpAddPermission(c.log, awsCfg, permission{
		Action:    "lambda:InvokeFunction",
		Principal: "logs." + awsCfg.Region + ".amazonaws.com",
	}))

	for _, trigger := range c.config.Triggers {
		executer.Add(newOpAddSubscriptionFilter(c.log, awsCfg, subscriptionFilter{
			LogGroupName:  trigger.LogGroupName,
			FilterName:    trigger.FilterName,
			FilterPattern: trigger.FilterPattern,
		}))
	}

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
	return nil
}
