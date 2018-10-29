// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"

	"github.com/elastic/beats/libbeat/logp"
)

type opWaitDeleteCloudFormation struct {
	log       *logp.Logger
	svc       *cloudformation.CloudFormation
	stackName string
}

func (o *opWaitDeleteCloudFormation) Execute() error {
	o.log.Debug("Waiting for cloudformation delete confirmation")
	status, reason, err := queryStackStatus(o.svc, o.stackName)

	// List of States from the cloud formation API.
	// https://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/API_Stack.html
	for {
		if err != nil {
			// Timing its possible that the stack doesn't exist at that point.
			if strings.Index(err.Error(), fmt.Sprintf("Stack with id %s does not exist", o.stackName)) != -1 {
				return nil
			}
			return err
		}

		o.log.Debugf(
			"Retrieving information on stack '%s' from cloudformation, current status: %v",
			o.stackName,
			*status,
		)

		switch *status {
		case cloudformation.StackStatusDeleteComplete: // OK
			return nil
		case cloudformation.StackStatusDeleteFailed:
			return fmt.Errorf("failed to delete the stack '%s', reason: %v", o.stackName, reason)
		case cloudformation.StackStatusRollbackFailed:
			return fmt.Errorf("failed to delete and rollback the stack '%s', reason: %v", o.stackName, reason)
		case cloudformation.StackStatusRollbackComplete:
			return fmt.Errorf("failed to delete the stack '%s', reason: %v", o.stackName, reason)
		}

		select {
		case <-time.After(periodicCheck):
			status, reason, err = queryStackStatus(o.svc, o.stackName)
		}
	}
}

func newWaitDeleteCloudFormation(log *logp.Logger, cfg aws.Config, stackName string) *opWaitDeleteCloudFormation {
	return &opWaitDeleteCloudFormation{log: log, svc: cloudformation.New(cfg), stackName: stackName}
}

func queryStackStatus(svc *cloudformation.CloudFormation, stackName string) (*cloudformation.StackStatus, *string, error) {
	input := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}
	req := svc.DescribeStacksRequest(input)
	resp, err := req.Send()
	if err != nil {
		return nil, nil, err
	}

	stack := resp.Stacks[0]
	return &stack.StackStatus, stack.StackStatusReason, nil
}
