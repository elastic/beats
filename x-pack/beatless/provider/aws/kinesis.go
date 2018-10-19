// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/beatless/core"
	"github.com/elastic/beats/x-pack/beatless/provider"
	"github.com/elastic/beats/x-pack/beatless/provider/aws/transformer"
)

// Kinesis receives events from a kinesis stream and forward them to elasticsearch.
type Kinesis struct {
	log *logp.Logger
}

// NewKinesis creates a new function to receives events from a kinesis stream.
func NewKinesis(provider provider.Provider, config *common.Config) (provider.Function, error) {
	return &Kinesis{log: logp.NewLogger("kinesis")}, nil
}

// Run starts the lambda function and wait for web triggers.
func (k *Kinesis) Run(_ context.Context, client core.Client) error {
	lambda.Start(k.createHandler(client))
	return nil
}

func (k *Kinesis) createHandler(client core.Client) func(request events.KinesisEvent) error {
	return func(request events.KinesisEvent) error {
		k.log.Debugf("The handler receives %d events", len(request.Records))

		events := transformer.KinesisEvent(request)
		if err := client.PublishAll(events); err != nil {
			k.log.Errorf("Could not publish events to the pipeline, error: %+v", err)
			return err
		}
		client.Wait()
		return nil
	}
}

// Name return the name of the lambda function.
func (k *Kinesis) Name() string {
	return "kinesis"
}
