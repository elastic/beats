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
	lambda.Start(func(request events.CloudwatchLogsData) error {
		c.log.Debug(
			"received %d events (logStream: %s, owner: %s, logGroup: %s, messageType: %s)",
			len(request.LogEvents),
			request.LogStream,
			request.Owner,
			request.LogGroup,
			request.MessageType,
		)

		// defensive checks
		if len(request.LogEvents) == 0 {
			c.log.Error("no log events received from cloudwatch log")
			return errors.New("no event received")
		}

		events := transformer.CloudwatchLogs(request)
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
