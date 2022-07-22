// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/x-pack/functionbeat/manager/executor"
	"github.com/elastic/elastic-agent-libs/logp"
)

type opUpdateCloudFormation struct {
	log         *logp.Logger
	svc         updateStackClient
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
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	}

	resp, err := o.svc.UpdateStack(context.TODO(), input)
	if err != nil {
		o.log.Debugf("Could not update the cloudformation stack, resp: %+v", resp)
		return err
	}

	c.ID = resp.StackId

	return nil
}

func newOpUpdateCloudFormation(
	log *logp.Logger,
	svc updateStackClient,
	templateURL, stackName string,
) *opUpdateCloudFormation {
	return &opUpdateCloudFormation{
		log:         log,
		svc:         svc,
		templateURL: templateURL,
		stackName:   stackName,
	}
}
