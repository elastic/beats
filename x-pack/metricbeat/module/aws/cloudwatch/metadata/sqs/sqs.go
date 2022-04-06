// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sqs

import (
	"context"
	"fmt"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

const metadataPrefix = "aws.sqs.queue"

// AddMetadata adds metadata for SQS queues from a specific region
func AddMetadata(endpoint string, regionName string, awsConfig awssdk.Config, fips_enabled bool, events map[string]mb.Event) map[string]mb.Event {
	sqsServiceName := awscommon.CreateServiceName("sqs", fips_enabled, regionName)
	svc := sqs.New(awscommon.EnrichAWSConfigWithEndpoint(
		endpoint, sqsServiceName, regionName, awsConfig))

	// Get queueUrls for each region
	queueURLs, err := getQueueUrls(svc)
	if err != nil {
		logp.Error(fmt.Errorf("getQueueUrls failed, skipping region %s: %w", regionName, err))
		return events
	}

	// collect monitoring state for each instance
	for _, queueURL := range queueURLs {
		queueURLParsed := strings.Split(queueURL, "/")
		queueName := queueURLParsed[len(queueURLParsed)-1]
		if _, ok := events[queueName]; !ok {
			continue
		}
		events[queueName].RootFields.Put(metadataPrefix+".name", queueName)
	}
	return events
}

func getQueueUrls(svc sqsiface.ClientAPI) ([]string, error) {
	// ListQueues
	listQueuesInput := &sqs.ListQueuesInput{}
	req := svc.ListQueuesRequest(listQueuesInput)
	output, err := req.Send(context.TODO())
	if err != nil {
		err = errors.Wrap(err, "Error ListQueues")
		return nil, err
	}
	return output.QueueUrls, nil
}
