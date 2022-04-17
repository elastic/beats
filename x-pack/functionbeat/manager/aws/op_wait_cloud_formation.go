// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"

	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/x-pack/functionbeat/manager/executor"
)

var periodicCheck = 2 * time.Second

type checkStatusFunc = func(*cloudformation.StackStatus) (bool, error)

type opWaitCloudFormation struct {
	log         *logp.Logger
	svc         cloudformationiface.ClientAPI
	checkStatus checkStatusFunc
}

func newOpWaitCloudFormation(
	log *logp.Logger,
	svc cloudformationiface.ClientAPI,
) *opWaitCloudFormation {
	return &opWaitCloudFormation{
		log:         log,
		svc:         svc,
		checkStatus: checkCreateStatus,
	}
}

func newWaitDeleteCloudFormation(
	log *logp.Logger,
	cfg aws.Config,
) *opWaitCloudFormation {
	return &opWaitCloudFormation{
		log:         log,
		svc:         cloudformation.New(cfg),
		checkStatus: checkDeleteStatus,
	}
}

func (o *opWaitCloudFormation) Execute(ctx executor.Context) error {
	c, ok := ctx.(*stackContext)
	if !ok {
		return errWrongContext
	}

	if c.ID == nil {
		return errMissingStackID
	}

	eventStackPoller := makeEventStackPoller(o.log, o.svc, periodicCheck, c)
	eventStackPoller.Start()
	defer eventStackPoller.Stop()

	for {
		status, _, err := queryStackStatus(o.svc, c.ID)
		if err != nil {
			return err
		}

		completed, err := o.checkStatus(status)
		if err != nil {
			return err
		}

		if completed {
			return nil
		}

		<-time.After(periodicCheck)
	}
}

func checkCreateStatus(status *cloudformation.StackStatus) (bool, error) {
	switch *status {
	case cloudformation.StackStatusUpdateComplete: // OK
		return true, nil
	case cloudformation.StackStatusCreateComplete: // OK
		return true, nil
	case cloudformation.StackStatusRollbackFailed:
		return true, errors.New("failed to create and rollback the stack")
	case cloudformation.StackStatusRollbackComplete:
		return true, errors.New("failed to create the stack")
	}
	return false, nil
}

func checkDeleteStatus(status *cloudformation.StackStatus) (bool, error) {
	switch *status {
	case cloudformation.StackStatusDeleteComplete: // OK
		return true, nil
	case cloudformation.StackStatusDeleteFailed:
		return true, errors.New("failed to delete the stack")
	case cloudformation.StackStatusRollbackFailed:
		return true, errors.New("failed to delete and rollback the stack")
	case cloudformation.StackStatusRollbackComplete:
		return true, errors.New("failed to delete the stack")
	}
	return false, nil
}

func queryStack(
	svc cloudformationiface.ClientAPI,
	stackID *string,
) (*cloudformation.DescribeStacksOutput, error) {
	input := &cloudformation.DescribeStacksInput{StackName: stackID}
	req := svc.DescribeStacksRequest(input)
	resp, err := req.Send(context.TODO())
	if err != nil {
		return nil, err
	}
	return resp.DescribeStacksOutput, nil
}

func queryStackStatus(
	svc cloudformationiface.ClientAPI,
	stackID *string,
) (*cloudformation.StackStatus, *string, error) {
	resp, err := queryStack(svc, stackID)
	if err != nil {
		return nil, nil, err
	}

	stack := resp.Stacks[0]
	return &stack.StackStatus, stack.StackStatusReason, nil
}

func queryStackID(svc cloudformationiface.ClientAPI, stackName *string) (*string, error) {
	resp, err := queryStack(svc, stackName)
	if err != nil {
		return nil, err
	}
	return resp.Stacks[0].StackId, nil
}
