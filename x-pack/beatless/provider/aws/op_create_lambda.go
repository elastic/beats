// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	lambdaApi "github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/elastic/beats/libbeat/logp"
)

var handlerName = "beatless"

type opCreateLambda struct {
	svc *lambdaApi.Lambda
	log *logp.Logger
}

func (o *opCreateLambda) Execute(ctx *executerContext) error {
	o.log.Debugf("create new lambda function with name: %s", ctx.Name)
	// Setup the environment to known which function to execute.
	envVariables := map[string]string{
		"BEAT_STRICT_PERMS": "false",
		"ENABLED_FUNCTIONS": ctx.Name,
	}

	req := &lambdaApi.CreateFunctionInput{
		Code:         &lambdaApi.FunctionCode{ZipFile: ctx.Content},
		FunctionName: aws.String(ctx.Name),
		Handler:      aws.String(ctx.HandleName),
		Role:         aws.String(ctx.Role),
		Runtime:      ctx.Runtime,
		Description:  aws.String(ctx.Description),
		Publish:      aws.Bool(true), // Create and publish a new version atomically.
		Environment:  &lambdaApi.Environment{Variables: envVariables},
	}

	api := o.svc.CreateFunctionRequest(req)
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

func (o *opCreateLambda) Rollback(ctx *executerContext) error {
	o.log.Debugf("remove lambda function with name: %s", ctx.Name)
	req := &lambdaApi.DeleteFunctionInput{FunctionName: aws.String(ctx.Name)}

	api := o.svc.DeleteFunctionRequest(req)
	resp, err := api.Send()
	if err != nil {
		o.log.Debugf("could not remove function, error: %s, response: %s", err, resp)
		return err
	}

	o.log.Debug("remove successful")
	return nil
}

func newOpCreateLambda(log *logp.Logger, awsCfg aws.Config) *opCreateLambda {
	if log == nil {
		log = logp.NewLogger("opCreateLambda")
	}
	return &opCreateLambda{log: log, svc: lambdaApi.New(awsCfg)}
}
