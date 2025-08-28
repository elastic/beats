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

	"github.com/elastic/beats/v7/metricbeat/mb"
)

const metadataPrefix = "aws.sqs.queue"

// AddMetadata adds metadata for SQS queues from a specific region
func AddMetadata(regionName string, awsConfig awssdk.Config, fips_enabled bool, events map[string]mb.Event) (map[string]mb.Event, error) {
	svc := sqs.NewFromConfig(awsConfig, func(o *sqs.Options) {
		if fips_enabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}

	})

	// Get queueUrls for each region
	queueURLs, err := getQueueUrls(svc)
	if err != nil {
		return events, fmt.Errorf("aws.sqs.queue fields are not available, skipping region %s: %w", regionName, err)
	}

	// collect monitoring state for each instance
	for _, queueURL := range queueURLs {
		queueURLParsed := strings.Split(queueURL, "/")
		queueName := queueURLParsed[len(queueURLParsed)-1]
		for eventIdentifier := range events {
			eventIdentifierComponents := strings.Split(eventIdentifier, "-")
			potentialQueueName := strings.Join(eventIdentifierComponents[0:len(eventIdentifierComponents)-1], "-")
			if queueName != potentialQueueName {
				continue
			}

			_, _ = events[eventIdentifier].RootFields.Put(metadataPrefix+".name", queueName)
		}
	}
	return events, nil
}

func getQueueUrls(svc *sqs.Client) ([]string, error) {
	// ListQueues
	listQueuesInput := &sqs.ListQueuesInput{}
	output, err := svc.ListQueues(context.TODO(), listQueuesInput)
	if err != nil {
		err = fmt.Errorf("error ListQueues: %w", err)
		return nil, err
	}
	return output.QueueUrls, nil
}
