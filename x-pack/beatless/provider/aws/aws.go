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
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/beatless/core"
	"github.com/elastic/beats/x-pack/beatless/provider"
	"github.com/elastic/beats/x-pack/beatless/provider/aws/transformer"
)

// Bundle exposes the trigger supported by the AWS provider.
var Bundle = provider.MustCreate(
	"aws",
	provider.NewDefaultProvider("aws"),
	feature.NewDetails("AWS Lambda", "listen to events on AWS lambda", feature.Experimental),
).MustAddFunction("cloudwatch_logs",
	NewCloudwatchLogs,
	feature.NewDetails(
		"Cloudwatch Logs trigger",
		"receive events from cloudwatch logs.",
		feature.Experimental,
	)).MustAddFunction("api_gateway_proxy",
	NewAPIGatewayProxy,
	feature.NewDetails(
		"API Gateway proxy trigger",
		"receive events from the api gateway proxy",
		feature.Experimental,
	)).MustAddFunction("kinesis",
	NewKinesis,
	feature.NewDetails(
		"Kinesis trigger",
		"receive events from a Kinesis stream",
		feature.Experimental,
	)).Bundle()

// Kinesis receives events from the web service and forward them to elasticsearch.
type Kinesis struct {
	log *logp.Logger
}

// NewKinesis creates a new function to receives events from a kinesis stream.
func NewKinesis(provider provider.Provider, config *common.Config) (provider.Function, error) {
	return &Kinesis{log: logp.NewLogger("kinesis")}, nil
}

// Run starts the lambda function and wait for web triggers.
func (k *Kinesis) Run(_ context.Context, client core.Client) error {
	lambda.Start(func(request events.KinesisEvent) error {
		k.log.Debug("received %d events", len(request.Records))

		// defensive checks
		if len(request.Records) == 0 {
			k.log.Error("no log events received from Kinesis")
			return errors.New("no event received")
		}

		events := transformer.KinesisEvent(request)
		if err := client.PublishAll(events); err != nil {
			k.log.Errorf("could not publish events to the pipeline, error: %s")
			return err
		}
		return nil
	})

	return nil
}

// Name return the name of the lambda function.
func (k *Kinesis) Name() string {
	return "kinesis"
}
