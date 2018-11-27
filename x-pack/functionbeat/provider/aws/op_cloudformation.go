// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"bytes"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/cloudformationiface"
	"github.com/gofrs/uuid"

	"github.com/elastic/beats/libbeat/logp"
)

type opCreateCloudFormation struct {
	log         *logp.Logger
	svc         cloudformationiface.CloudFormationAPI
	templateURL string
	stackName   string
}

func newOpCreateCloudFormation(
	log *logp.Logger,
	svc cloudformationiface.CloudFormationAPI,
	templateURL, stackName string,
) *opCreateCloudFormation {
	return &opCreateCloudFormation{
		log:         log,
		svc:         svc,
		templateURL: templateURL,
		stackName:   stackName,
	}
}

func (o *opCreateCloudFormation) Execute(ctx executionContext) error {
	c, ok := ctx.(*stackContext)
	if !ok {
		return errWrongContext
	}

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
		o.log.Debugf("Could not create the CloudFormation stack request, resp: %v", resp)
		return err
	}

	c.ID = resp.StackId

	return nil
}

func makeEventStackPoller(
	log *logp.Logger,
	svc cloudformationiface.CloudFormationAPI,
	periodicCheck time.Duration,
	ctx *stackContext,
) *eventStackPoller {
	return newEventStackPoller(
		log,
		svc,
		ctx.ID,
		periodicCheck,
		&reportStackEvent{skipBefore: ctx.StartedAt, callback: func(event cloudformation.StackEvent) {
			// Returned values for a stack events are hit or miss, so lets try to create a
			// meaningful string.
			var buf bytes.Buffer

			buf.WriteString("Stack event received")
			if event.ResourceType != nil {
				buf.WriteString(", ResourceType: ")
				buf.WriteString(*event.ResourceType)
			}

			if event.LogicalResourceId != nil {
				buf.WriteString(", LogicalResourceId: ")
				buf.WriteString(*event.LogicalResourceId)
			}

			buf.WriteString(", ResourceStatus: ")
			buf.WriteString(string(event.ResourceStatus))

			if event.ResourceStatusReason != nil {
				buf.WriteString(", ResourceStatusReason: ")
				buf.WriteString(*event.ResourceStatusReason)
			}

			log.Info(buf.String())
		}},
	)
}
