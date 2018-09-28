// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	lambdaApi "github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/elastic/beats/libbeat/logp"
)

var aliasSuffix = "PROD"

type opCreateAlias struct {
	svc *lambdaApi.Lambda
	log *logp.Logger
}

func (o *opCreateAlias) Execute(ctx *executorContext) error {
	o.log.Debugf("creating new alias for function with name: %s", ctx.Name)
	req := &lambdaApi.CreateAliasInput{
		Description:     aws.String("alias for " + ctx.Name),
		FunctionVersion: aws.String("$LATEST"),
		FunctionName:    aws.String(ctx.Name),
		Name:            aws.String(aliasSuffix),
	}

	api := o.svc.CreateAliasRequest(req)
	resp, err := api.Send()
	if err != nil {
		o.log.Debugf("could not create alias, error: %s, response: %s", err, resp)
		return err
	}

	ctx.AliasArn = *resp.AliasArn
	o.log.Debug("alias created successfully")
	return nil
}

func (o *opCreateAlias) Rollback(ctx executorContext) error {
	o.log.Debugf("remove alias for function with name: %s", ctx.Name)

	req := &lambdaApi.DeleteAliasInput{
		FunctionName: aws.String(ctx.Name),
		Name:         aws.String(aliasSuffix),
	}

	api := o.svc.DeleteAliasRequest(req)
	resp, err := api.Send()
	if err != nil {
		o.log.Debugf("could not remove alias, error: %s, response: %s", err, resp)
		return err
	}

	o.log.Debug("alias removed successfully")
	return nil
}

func newOpCreateAlias(log *logp.Logger, awsCfg aws.Config) *opCreateAlias {
	if log == nil {
		log = logp.NewLogger("opCreateAlias")
	}
	return &opCreateAlias{log: log, svc: lambdaApi.New(awsCfg)}
}
