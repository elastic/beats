// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	lambdaApi "github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/elastic/beats/libbeat/logp"
)

type permission struct {
	Action    string
	Principal string
}

type opAddPermission struct {
	svc        *lambdaApi.Lambda
	log        *logp.Logger
	permission permission
}

func (o *opAddPermission) Execute(ctx *executorContext) error {
	o.log.Debugf(
		"adding permissions, action: %s, principal: %s",
		o.permission.Action,
		o.permission.Principal,
	)

	if ctx.AliasArn == "" {
		return errors.New("AliasArn not found, need to create or update the alias")
	}

	permissions := &lambdaApi.AddPermissionInput{
		Action:       aws.String(o.permission.Action),
		Principal:    aws.String(o.permission.Principal),
		FunctionName: aws.String(ctx.AliasArn),
		StatementId:  aws.String(strconv.Itoa(int(time.Now().Unix()))),
		// 		// SourceArn: // must be the cloudwatch arn
	}

	permissionsSend := o.svc.AddPermissionRequest(permissions)
	permissionResp, err := permissionsSend.Send()
	if err != nil {
		o.log.Debugf("could not add permission to function, error: %s, response: %s", err, permissionResp)
		return err
	}

	o.log.Debugf("adding permissions successful")
	return nil
}

func newOpAddPermission(log *logp.Logger, awsCfg aws.Config, permission permission) *opAddPermission {
	if log == nil {
		log = logp.NewLogger("opAddPermission")
	}
	return &opAddPermission{log: log, svc: lambdaApi.New(awsCfg), permission: permission}
}
