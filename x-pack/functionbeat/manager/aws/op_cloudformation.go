// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/functionbeat/manager/executor"
)

type opCreateCloudFormation struct {
	log         *logp.Logger
	svc         cloudformationiface.ClientAPI
	templateURL string
	stackName   string
}

func newOpCreateCloudFormation(
	log *logp.Logger,
	svc cloudformationiface.ClientAPI,
	templateURL, stackName string,
) *opCreateCloudFormation {
	return &opCreateCloudFormation{
		log:         log,
		svc:         svc,
		templateURL: templateURL,
		stackName:   stackName,
	}
}

func (o *opCreateCloudFormation) Execute(ctx executor.Context) error {
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
	resp, err := req.Send(context.TODO())
	if err != nil {
		o.log.Debugf("Could not create the CloudFormation stack request, resp: %v", resp)
		return err
	}

	c.ID = resp.StackId

	return nil
}

func makeEventStackPoller(
	log *logp.Logger,
	svc cloudformationiface.ClientAPI,
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
			var buf strings.Builder
			fmt.Fprintf(&buf, "Stack event received")

			writeOptKV(&buf, "ResourceType", event.ResourceType)
			writeOptKV(&buf, "LogicalResourceId", event.LogicalResourceId)
			s := string(event.ResourceStatus)
			writeOptKV(&buf, "ResourceStatus", &s)
			writeOptKV(&buf, "ResourceStatusReason", event.ResourceStatusReason)

			log.Info(buf.String())
		}},
	)
}

func writeKV(buf *strings.Builder, key string, value string) {
	fmt.Fprintf(buf, ", %s: %s", key, value)
}

func writeOptKV(buf *strings.Builder, key string, value *string) {
	if value != nil {
		writeKV(buf, key, *value)
	}
}
