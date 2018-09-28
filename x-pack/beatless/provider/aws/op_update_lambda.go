// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	lambdaApi "github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/elastic/beats/libbeat/logp"
)

type opUpdateLambda struct {
	svc *lambdaApi.Lambda
	log *logp.Logger
}

func (o *opUpdateLambda) Execute(ctx *executorContext) error {
	o.log.Debugf("updating lambda function with name: %s", ctx.Name)

	req := &lambdaApi.UpdateFunctionCodeInput{
		ZipFile:      ctx.Content,
		FunctionName: aws.String(ctx.Name),
		Publish:      aws.Bool(true), // Create and publish a new version.
	}

	api := o.svc.UpdateFunctionCodeRequest(req)
	resp, err := api.Send()
	if err != nil {
		o.log.Debugf("could not create function, error: %s, response: %s", err, resp)
		return err
	}

	// retrieve the function arn for future calls.
	ctx.FunctionArn = *resp.FunctionArn

	o.log.Debug("creation successful")
	return nil
}

func newOpUpdateLambda(log *logp.Logger, awsCfg aws.Config) *opUpdateLambda {
	if log == nil {
		log = logp.NewLogger("opUpdateLambda")
	}

	return &opUpdateLambda{log: log, svc: lambdaApi.New(awsCfg)}
}
