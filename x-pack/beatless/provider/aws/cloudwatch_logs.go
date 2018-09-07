// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/beatless/core"
	"github.com/elastic/beats/x-pack/beatless/provider"
	"github.com/elastic/beats/x-pack/beatless/provider/aws/transformer"
)

// CloudwatchLogs receives CloudwatchLogs events from a lambda function and forward the logs to
// an Elasticsearch cluster.
type CloudwatchLogs struct {
	log *logp.Logger
}

// NewCloudwatchLogs create a new function to listen to cloudwatch logs events.
func NewCloudwatchLogs(provider provider.Provider, config *common.Config) (provider.Function, error) {
	return &CloudwatchLogs{log: logp.NewLogger("cloudwatch_logs")}, nil
}

// Run start the AWS lambda handles and will transform any events received to the pipeline.
func (c *CloudwatchLogs) Run(_ context.Context, client core.Client) error {
	lambda.Start(func(request events.CloudwatchLogsEvent) error {
		parsedEvent, err := request.AWSLogs.Parse()
		if err != nil {
			c.log.Errorf("could not parse events from cloudwatch logs, error: %s", err)
			return err
		}

		// defensive checks
		if len(parsedEvent.LogEvents) == 0 {
			c.log.Error("no log events received from cloudwatch log")
			return errors.New("no event received")
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
		return nil
	})
	return nil
}

// Name returns the name of the function.
func (c CloudwatchLogs) Name() string {
	return "cloudwatch_logs"
}
