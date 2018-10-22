// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transformer

import (
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// Centralize anything related to ECS into a common file.

// CloudwatchLogs takes an CloudwatchLogsData and transform it into a beat event.
func CloudwatchLogs(request events.CloudwatchLogsData) []beat.Event {
	events := make([]beat.Event, len(request.LogEvents))

	for idx, logEvent := range request.LogEvents {
		ts := time.Unix(0, logEvent.Timestamp*int64(time.Millisecond)).UTC()
		events[idx] = beat.Event{
			Timestamp: ts,
			Fields: common.MapStr{
				"read_timestamp":                         time.Now(),
				"event.id":                               logEvent.ID,
				"message":                                logEvent.Message,
				"user.id":                                request.Owner,
				"aws.cloudwatch.log_stream":              request.LogStream,
				"aws.cloudwatch.log_group":               request.LogGroup,
				"aws.cloudwatchlog.message_type":         request.MessageType,
				"aws.cloudwatchlog.subscription_filters": request.SubscriptionFilters,
			},
		}
	}

	return events
}

// APIGatewayProxy takes a web request on the api gateway proxy and transform it into a beat event.
func APIGatewayProxy(request events.APIGatewayProxyRequest) beat.Event {
	return beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"event.id":                                request.RequestContext.RequestID,
			"read_timestamp":                          time.Now(),
			"message":                                 request.Body,
			"aws.api_gateway_proxy.resource":          request.Resource,
			"aws.api_gateway_proxy.path":              request.Path,
			"aws.api_gateway_proxy.method":            request.HTTPMethod,
			"aws.api_gateway_proxy.headers":           request.Headers,
			"aws.api_gateway_proxy.query_string":      request.QueryStringParameters,
			"aws.api_gateway_proxy.path_parameters":   request.PathParameters,
			"aws.api_gateway_proxy.is_base64_encoded": request.IsBase64Encoded,
		},
	}
}

// Kinesis takes a kinesis event and create multiples beat events.
func Kinesis(request events.KinesisEvent) []beat.Event {
	events := make([]beat.Event, len(request.Records))
	for idx, record := range request.Records {
		events[idx] = beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"event.id":             record.EventID,
				"read_timestamp":       time.Now(),
				"cloud.region":         record.AwsRegion,
				"message":              record.Kinesis.Data,
				"aws.event_name":       record.EventName,
				"aws.event_source":     record.EventSource,
				"aws.event_source_arn": record.EventSourceArn,
				"aws.kinesis.version":  record.EventVersion,
			},
		}
	}
	return events
}

// SQS takes a SQS event and create multiples beat events.
func SQS(request events.SQSEvent) []beat.Event {
	events := make([]beat.Event, len(request.Records))
	for idx, record := range request.Records {
		events[idx] = beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"event.id":               record.MessageId,
				"aws.event_source":       record.EventSource,
				"aws.event_source_arn":   record.EventSourceARN,
				"read_timestamp":         time.Now(),
				"cloud.region":           record.AWSRegion,
				"aws.sqs.receipt_handle": record.ReceiptHandle,
				"aws.sqs.attributes":     record.Attributes,
				"message":                record.Body,
			},
		}
	}
	return events
}
