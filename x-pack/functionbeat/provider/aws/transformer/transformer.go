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
// TODO: Look at the fields to align them with ECS.
// TODO: how to keep the fields in sync with AWS?
// TODO: api gateway proxy a lot more information is available.

// CloudwatchLogs takes an CloudwatchLogsData and transform it into a beat event.
func CloudwatchLogs(request events.CloudwatchLogsData) []beat.Event {
	events := make([]beat.Event, len(request.LogEvents))

	for idx, logEvent := range request.LogEvents {
		ts := time.Unix(0, logEvent.Timestamp*int64(time.Millisecond)).UTC()
		events[idx] = beat.Event{
			Timestamp: ts,
			Fields: common.MapStr{
				"functionbeat.aws.id":                             logEvent.ID,
				"message":                                         logEvent.Message,
				"functionbeat.cloudwatchlog.owner":                request.Owner,
				"functionbeat.cloudwatch.log_stream":              request.LogStream,
				"functionbeat.cloudwatch.log_group":               request.LogGroup,
				"functionbeat.cloudwatchlog.message_type":         request.MessageType,
				"functionbeat.cloudwatchlog.subscription_filters": request.SubscriptionFilters,
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
			"functionbeat.aws.id":                            request.RequestContext.RequestID,
			"functionbeat.api_gateway_proxy.resource":        request.Resource,
			"functionbeat.api_gateway_proxy.path":            request.Path,
			"functionbeat.api_gateway_proxy.method":          request.HTTPMethod,
			"functionbeat.api_gateway_proxy.headers":         request.Headers,
			"functionbeat.api_gateway_proxy.query_string":    request.QueryStringParameters,
			"functionbeat.api_gateway_proxy.path_parameters": request.PathParameters,
			"message":                         request.Body,
			"function.beat.is_base64_encoded": request.IsBase64Encoded,
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
				"functionbeat.aws": common.MapStr{
					"id":               record.EventID,
					"event_name":       record.EventName,
					"event_source":     record.EventSource,
					"event_source_arn": record.EventSourceArn,
					"region":           record.AwsRegion,
				},
				"message": record.Kinesis.Data,
				"functionbeat.kinesis.event_version": record.EventVersion,
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
				"functionbeat.aws": common.MapStr{
					"id":               record.MessageId,
					"event_source":     record.EventSource,
					"event_source_arn": record.EventSourceARN,
					"region":           record.AWSRegion,
				},
				"functionbeat.sqs.receipt_handle": record.ReceiptHandle,
				"functionbeat.sqs.attributes":     record.Attributes,
				"message":                         record.Body,
			},
		}
	}
	return events
}
