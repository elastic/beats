// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/gofrs/uuid"

	"github.com/elastic/beats/libbeat/logp"
)

type opDeleteCloudFormation struct {
	log       *logp.Logger
	svc       *cloudformation.CloudFormation
	stackName string
}

func (o *opDeleteCloudFormation) Execute() error {
	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}
	input := &cloudformation.DeleteStackInput{
		ClientRequestToken: aws.String(uuid.String()),
		StackName:          aws.String(o.stackName),
	}

	req := o.svc.DeleteStackRequest(input)
	resp, err := req.Send()
	if err != nil {
		o.log.Debugf("Could not delete the stack, response: %v", resp)
		return err
	}
	return nil
}

func newOpDeleteCloudFormation(
	log *logp.Logger,
	cfg aws.Config,
	stackName string,
) *opDeleteCloudFormation {
	return &opDeleteCloudFormation{log: log, svc: cloudformation.New(cfg), stackName: stackName}
}
