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
	"github.com/elastic/beats/x-pack/functionbeat/core"
	"github.com/elastic/beats/x-pack/functionbeat/provider"
	"github.com/elastic/beats/x-pack/functionbeat/provider/aws/transformer"
)

// SQS receives events from the web service and forward them to elasticsearch.
type SQS struct {
	log *logp.Logger
}

// NewSQS creates a new function to receives events from a SQS queue.
func NewSQS(provider provider.Provider, config *common.Config) (provider.Function, error) {
	return &SQS{log: logp.NewLogger("sqs")}, nil
}

// Run starts the lambda function and wait for web triggers.
func (s *SQS) Run(_ context.Context, client core.Client) error {
	lambda.Start(s.createHandler(client))
	return nil
}

func (s *SQS) createHandler(client core.Client) func(request events.SQSEvent) error {
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
