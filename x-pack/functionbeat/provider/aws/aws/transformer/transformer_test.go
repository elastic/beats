// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transformer

import (
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestCloudwatch(t *testing.T) {
	logsData := events.CloudwatchLogsData{
		Owner:               "me",
		LogGroup:            "my-group",
		LogStream:           "stream",
		SubscriptionFilters: []string{"MyFilter"},
		MessageType:         "DATA_MESSAGE",
		LogEvents: []events.CloudwatchLogsLogEvent{
			events.CloudwatchLogsLogEvent{
				ID:        "1234567890123456789",
				Timestamp: 1566908691193,
				Message:   "my interesting message",
			},
		},
	}

	events := CloudwatchLogs(logsData)
	assert.Equal(t, 1, len(events))

	currentLoc, err := time.LoadLocation("Local")
	assert.Nil(t, err)

	expectedTime, err := time.ParseInLocation(time.RFC3339, "2019-08-27T14:24:51.193+02:00", currentLoc)
	assert.Nil(t, err)

	expectedEvent := beat.Event{
		Timestamp: expectedTime,
		Fields: common.MapStr{
			"message":              "my interesting message",
			"id":                   "1234567890123456789",
			"owner":                "me",
			"log_stream":           "stream",
			"log_group":            "my-group",
			"message_type":         "DATA_MESSAGE",
			"subscription_filters": []string{"MyFilter"},
		},
	}

	assert.Equal(t, expectedEvent.Fields, events[0].Fields)
	assert.Equal(t, expectedEvent.Timestamp, events[0].Timestamp)
}

func TestKinesis(t *testing.T) {
	request := events.KinesisEvent{
		Records: []events.KinesisEventRecord{
			events.KinesisEventRecord{
				AwsRegion:      "us-east-1",
				EventID:        "1234",
				EventName:      "connect",
				EventSource:    "web",
				EventVersion:   "1.0",
				EventSourceArn: "arn:aws:iam::00000000:role/functionbeat",
				Kinesis: events.KinesisRecord{
					Data:                 []byte("hello world"),
					PartitionKey:         "abc123",
					SequenceNumber:       "12345",
					KinesisSchemaVersion: "1.0",
					EncryptionType:       "test",
				},
			},
		},
	}

	events := KinesisEvent(request)
	assert.Equal(t, 1, len(events))

	fields := common.MapStr{
		"event_id":                "1234",
		"event_name":              "connect",
		"event_source":            "web",
		"event_source_arn":        "arn:aws:iam::00000000:role/functionbeat",
		"event_version":           "1.0",
		"aws_region":              "us-east-1",
		"message":                 "hello world",
		"kinesis_partition_key":   "abc123",
		"kinesis_schema_version":  "1.0",
		"kinesis_sequence_number": "12345",
		"kinesis_encryption_type": "test",
	}

	assert.Equal(t, fields, events[0].Fields)
}
