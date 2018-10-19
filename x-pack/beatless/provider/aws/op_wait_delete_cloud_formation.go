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
	status, _, err := queryStackStatus(o.svc, o.stackName)

	for err == nil && strings.Index(string(*status), "FAILED") == -1 {
		select {
		case <-time.After(periodicCheck):
			status, _, err = queryStackStatus(o.svc, o.stackName)
		}
	}

	// Since most of the type used by the AWS framework are generated from a schema definition
	// I have no other way to detect that the stack is deleted.
	if strings.Index(err.Error(), "Stack with id "+o.stackName+" does not exist") != -1 {
		return nil
	}

	if err != nil {
		return err
	}

	return nil
}

func newWaitDeleteCloudFormation(log *logp.Logger, cfg aws.Config, stackName string) *opWaitDeleteCloudFormation {
	return &opWaitDeleteCloudFormation{log: log, svc: cloudformation.New(cfg), stackName: stackName}
}

func queryStackStatus(svc *cloudformation.CloudFormation, stackName string) (*cloudformation.StackStatus, string, error) {
	input := &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}
	req := svc.DescribeStacksRequest(input)
	resp, err := req.Send()
	if err != nil {
		return nil, "", err
	}

	if len(resp.Stacks) == 0 {
		return nil, "", fmt.Errorf("no stack found with the name %s", stackName)
	}

	stack := resp.Stacks[0]
	return &stack.StackStatus, "", nil
}
