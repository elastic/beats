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

type opCreateCloudFormation struct {
	log         *logp.Logger
	svc         *cloudformation.CloudFormation
	templateURL string
	stackName   string
}

func newOpCreateCloudFormation(
	log *logp.Logger,
	cfg aws.Config,
	templateURL, stackName string,
) *opCreateCloudFormation {
	return &opCreateCloudFormation{
		log:         log,
		svc:         cloudformation.New(cfg),
		templateURL: templateURL,
		stackName:   stackName,
	}
}

func (o *opCreateCloudFormation) Execute() error {
	o.log.Debug("Creating CloudFormation create stack request")
	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}
	input := &cloudformation.CreateStackInput{
		ClientRequestToken: aws.String(uuid.String()),
		StackName:          aws.String(o.stackName),
		TemplateURL:        aws.String(o.templateURL),
		Capabilities: []cloudformation.Capability{
			cloudformation.CapabilityCapabilityNamedIam,
		},
	}

	req := o.svc.CreateStackRequest(input)
	resp, err := req.Send()
	if err != nil {
		o.log.Debugf("Could not create the cloud formation stack request, resp: %v", resp)
		return err
	}
	return nil
}
