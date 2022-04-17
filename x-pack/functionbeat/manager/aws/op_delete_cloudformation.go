// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"
	"github.com/gofrs/uuid"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/x-pack/functionbeat/manager/executor"
)

type opDeleteCloudFormation struct {
	log       *logp.Logger
	svc       cloudformationiface.ClientAPI
	stackName string
}

func (o *opDeleteCloudFormation) Execute(ctx executor.Context) error {
	c, ok := ctx.(*stackContext)
	if !ok {
		return errWrongContext
	}

	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}

	// retrieve the stack id from the name so we can have access to the Stack events when the
	// stack is completely deleted.
	stackID, err := queryStackID(o.svc, aws.String(o.stackName))
	if err != nil {
		return err
	}
	c.ID = stackID

	input := &cloudformation.DeleteStackInput{
		ClientRequestToken: aws.String(uuid.String()),
		StackName:          aws.String(o.stackName),
	}

	req := o.svc.DeleteStackRequest(input)
	resp, err := req.Send(context.TODO())
	if err != nil {
		o.log.Debugf("Could not delete the stack, response: %v", resp)
		return err
	}

	return nil
}

func newOpDeleteCloudFormation(
	log *logp.Logger,
	svc cloudformationiface.ClientAPI,
	stackName string,
) *opDeleteCloudFormation {
	return &opDeleteCloudFormation{log: log, svc: svc, stackName: stackName}
}
