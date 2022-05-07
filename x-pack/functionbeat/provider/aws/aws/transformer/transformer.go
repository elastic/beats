// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transformer

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/awslabs/kinesis-aggregation/go/deaggregator"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Centralize anything related to ECS into a common file.
// TODO: Look at the fields to align them with ECS.
// TODO: how to keep the fields in sync with AWS?
// TODO: api gateway proxy a lot more information is available.

// CloudwatchLogs takes an CloudwatchLogsData and transform it into a beat event.
func CloudwatchLogs(request events.CloudwatchLogsData) []beat.Event {
	events := make([]beat.Event, len(request.LogEvents))

	for idx, logEvent := range request.LogEvents {
		events[idx] = beat.Event{
			Timestamp: time.Unix(0, logEvent.Timestamp*1000000),
			Fields: mapstr.M{
				"event": mapstr.M{
					"kind": "event",
				},
				"cloud": mapstr.M{
					"provider": "aws",
				},
				"message":              logEvent.Message,
				"id":                   logEvent.ID,
				"owner":                request.Owner,
				"log_stream":           request.LogStream,
				"log_group":            request.LogGroup,
				"message_type":         request.MessageType,
				"subscription_filters": request.SubscriptionFilters,
			},
		}
	}

	return events
}

// APIGatewayProxyRequest takes a web request on the api gateway proxy and transform it into a beat event.
func APIGatewayProxyRequest(request events.APIGatewayProxyRequest) beat.Event {
	return beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"event": mapstr.M{
				"kind":     "event",
				"category": []string{"network"},
				"type":     []string{"connection", "protocol"},
			},
			"cloud": mapstr.M{
				"provider":   "aws",
				"account.id": request.RequestContext.AccountID,
			},
			"network": mapstr.M{
				"transport": "tcp",
				"protocol":  "http",
			},
			"resource":          request.Resource,
			"path":              request.Path,
			"method":            request.HTTPMethod,
			"headers":           request.Headers,               // TODO: ECS map[string]
			"query_string":      request.QueryStringParameters, // TODO: map[string], might conflict with ECS
			"path_parameters":   request.PathParameters,
			"body":              request.Body, // TODO: could be JSON, json processor? could be used by other functions.
			"is_base64_encoded": request.IsBase64Encoded,
		},
	}
}

// KinesisEvent takes a kinesis event and create multiples beat events.
// DOCS: https://docs.aws.amazon.com/lambda/latest/dg/with-kinesis.html
func KinesisEvent(request events.KinesisEvent) ([]beat.Event, error) {
	var events []beat.Event
	for _, record := range request.Records {
		kr := &kinesis.Record{
			ApproximateArrivalTimestamp: &record.Kinesis.ApproximateArrivalTimestamp.Time,
			Data:                        record.Kinesis.Data,
			EncryptionType:              &record.Kinesis.EncryptionType,
			PartitionKey:                &record.Kinesis.PartitionKey,
			SequenceNumber:              &record.Kinesis.SequenceNumber,
		}
		deaggRecords, err := deaggregator.DeaggregateRecords([]*kinesis.Record{kr})
		if err != nil {
			return nil, err
		}

		for _, deaggRecord := range deaggRecords {
			events = append(events, beat.Event{
				Timestamp: time.Now(),
				Fields: mapstr.M{
					"event": mapstr.M{
						"kind": "event",
					},
					"cloud": mapstr.M{
						"provider": "aws",
						"region":   record.AwsRegion,
					},
					"event_id":                record.EventID,
					"event_name":              record.EventName,
					"event_source":            record.EventSource,
					"event_source_arn":        record.EventSourceArn,
					"event_version":           record.EventVersion,
					"aws_region":              record.AwsRegion,
					"message":                 string(deaggRecord.Data),
					"kinesis_partition_key":   *deaggRecord.PartitionKey,
					"kinesis_schema_version":  record.Kinesis.KinesisSchemaVersion,
					"kinesis_sequence_number": *deaggRecord.SequenceNumber,
					"kinesis_encryption_type": *deaggRecord.EncryptionType,
				},
			})
		}
	}
	return events, nil
}

// CloudwatchKinesisEvent takes a Kinesis event containing Cloudwatch logs and creates events for all
// Cloudwatch logs.
func CloudwatchKinesisEvent(request events.KinesisEvent, base64Encoded, compressed bool) ([]beat.Event, error) {
	var evts []beat.Event
	for _, record := range request.Records {
		envelopeFields := mapstr.M{
			"event": mapstr.M{
				"kind": "event",
			},
			"cloud": mapstr.M{
				"provider": "aws",
				"region":   record.AwsRegion,
			},
			"event_id":                record.EventID,
			"event_name":              record.EventName,
			"event_source":            record.EventSource,
			"event_source_arn":        record.EventSourceArn,
			"event_version":           record.EventVersion,
			"aws_region":              record.AwsRegion,
			"kinesis_partition_key":   record.Kinesis.PartitionKey,
			"kinesis_schema_version":  record.Kinesis.KinesisSchemaVersion,
			"kinesis_sequence_number": record.Kinesis.SequenceNumber,
			"kinesis_encryption_type": record.Kinesis.EncryptionType,
		}

		kinesisData := record.Kinesis.Data
		if base64Encoded {
			var err error
			kinesisData, err = base64.StdEncoding.DecodeString(string(kinesisData))
			if err != nil {
				return nil, err
			}
		}

		if compressed {
			inBuf := bytes.NewBuffer(record.Kinesis.Data)
			r, err := gzip.NewReader(inBuf)
			if err != nil {
				return nil, err
			}

			var outBuf bytes.Buffer
			_, err = io.Copy(&outBuf, r)
			if err != nil {
				r.Close()
				return nil, err
			}

			err = r.Close()
			if err != nil {
				return nil, err
			}
			kinesisData = outBuf.Bytes()
		}

		var cloudwatchEvents events.CloudwatchLogsData
		err := json.Unmarshal(kinesisData, &cloudwatchEvents)
		if err != nil {
			return nil, err
		}

		cwEvts := CloudwatchLogs(cloudwatchEvents)
		for _, cwe := range cwEvts {
			cwe.Fields.DeepUpdate(envelopeFields)
			evts = append(evts, cwe)
		}
	}
	return evts, nil
}

// SQS takes a SQS event and create multiples beat events.
func SQS(request events.SQSEvent) []beat.Event {
	events := make([]beat.Event, len(request.Records))
	for idx, record := range request.Records {
		events[idx] = beat.Event{
			Timestamp: time.Now(),
			Fields: mapstr.M{
				"event": mapstr.M{
					"kind": "event",
				},
				"cloud": mapstr.M{
					"provider": "aws",
					"region":   record.AWSRegion,
				},
				"message_id":       record.MessageId,
				"receipt_handle":   record.ReceiptHandle,
				"message":          record.Body,
				"attributes":       record.Attributes,
				"event_source":     record.EventSource,
				"event_source_arn": record.EventSourceARN,
				"aws_region":       record.AWSRegion,
			},
			// TODO: SQS message attributes missing, need to check doc
		}
	}
	return events
}
