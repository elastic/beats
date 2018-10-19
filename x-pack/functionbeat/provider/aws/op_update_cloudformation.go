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

type opUpdateCloudFormation struct {
	log         *logp.Logger
	svc         *cloudformation.CloudFormation
	templateURL string
	stackName   string
}

func (o *opUpdateCloudFormation) Execute() error {
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
	resp, err := req.Send()
	if err != nil {
		o.log.Debug("Could not update the cloudformation stack, resp: %s", resp)
		return err
	}
	return nil
}

func newOpUpdateCloudFormation(
	log *logp.Logger,
	cfg aws.Config,
	templateURL, stackName string,
) *opUpdateCloudFormation {
	return &opUpdateCloudFormation{
		log:         log,
		svc:         cloudformation.New(cfg),
		templateURL: templateURL,
		stackName:   stackName,
	}
}
