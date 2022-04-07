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

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/x-pack/functionbeat/manager/executor"
)

type opUpdateCloudFormation struct {
	log         *logp.Logger
	svc         cloudformationiface.ClientAPI
	templateURL string
	stackName   string
}

func (o *opUpdateCloudFormation) Execute(ctx executor.Context) error {
	c, ok := ctx.(*stackContext)
	if !ok {
		return errWrongContext
	}

	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}
	input := &cloudformation.UpdateStackInput{
		ClientRequestToken: aws.String(uuid.String()),
		StackName:          aws.String(o.stackName),
		TemplateURL:        aws.String(o.templateURL),
		Capabilities: []cloudformation.Capability{
			cloudformation.CapabilityCapabilityNamedIam,
		},
	}

	req := o.svc.UpdateStackRequest(input)
	resp, err := req.Send(context.TODO())
	if err != nil {
		o.log.Debugf("Could not update the cloudformation stack, resp: %+v", resp)
		return err
	}

	c.ID = resp.StackId

	return nil
}

func newOpUpdateCloudFormation(
	log *logp.Logger,
	svc cloudformationiface.ClientAPI,
	templateURL, stackName string,
) *opUpdateCloudFormation {
	return &opUpdateCloudFormation{
		log:         log,
		svc:         svc,
		templateURL: templateURL,
		stackName:   stackName,
	}
}
