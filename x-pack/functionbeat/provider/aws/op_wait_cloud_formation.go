// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"

	"github.com/elastic/beats/libbeat/logp"
)

var periodicCheck = 10 * time.Second

type opCloudWaitCloudFormation struct {
	log       *logp.Logger
	svc       *cloudformation.CloudFormation
	stackName string
}

func newOpWaitCloudFormation(
	log *logp.Logger,
	cfg aws.Config,
	stackName string,
) *opCloudWaitCloudFormation {
	return &opCloudWaitCloudFormation{
		log:       log,
		svc:       cloudformation.New(cfg),
		stackName: stackName,
	}
}

func (o *opCloudWaitCloudFormation) Execute() error {
	o.log.Debug("Waiting for cloudformation confirmation")
	status, reason, err := queryStackStatus(o.svc, o.stackName)

	// List of States from the cloud formation API.
	// https://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/API_Stack.html
	for {
		o.log.Debugf(
			"Retrieving information on stack '%s' from cloudformation, current status: %v",
			o.stackName,
			*status,
		)

		if err != nil {
			return err
		}

		switch *status {
		case cloudformation.StackStatusUpdateComplete: // OK
			return nil
		case cloudformation.StackStatusCreateComplete: // OK
			return nil
		case cloudformation.StackStatusCreateFailed:
			return fmt.Errorf("failed to create the stack '%s', reason: %v", o.stackName, reason)
		case cloudformation.StackStatusRollbackFailed:
			return fmt.Errorf("failed to create and rollback the stack '%s', reason: %v", o.stackName, reason)
		case cloudformation.StackStatusRollbackComplete:
			return fmt.Errorf("failed to create the stack '%s', reason: %v", o.stackName, reason)
		}

		select {
		case <-time.After(periodicCheck):
			status, reason, err = queryStackStatus(o.svc, o.stackName)
		}
	}
}
