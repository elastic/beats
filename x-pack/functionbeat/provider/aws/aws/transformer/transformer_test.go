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

	expectedTime, err := time.ParseInLocation(time.RFC3339, "2019-08-27T12:24:51.193+00:00", time.UTC)
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
	assert.Equal(t, expectedEvent.Timestamp, events[0].Timestamp.UTC())
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

func TestCloudwatchKinesis(t *testing.T) {
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
					Data: []byte(`eyJtZXNzYWdlVHlwZSI6IkRBVEFfTUVTU0FHRSIsIm93bmVyIjoiMDc5NzA5NzYxMTUyIiwibG9n
R3JvdXAiOiJ0ZXN0ZW0iLCJsb2dTdHJlYW0iOiJmb2x5b21hbnkiLCJzdWJzY3JpcHRpb25GaWx0
ZXJzIjpbIk1pbmRlbiJdLCJsb2dFdmVudHMiOlt7ImlkIjoiMzQ5MzM1ODk4NzM5NzIwNDAzMDgy
ODAzMDIzMTg1MjMxODU5NjA1NTQxODkxNjg4NzMyNDI2MjQiLCJ0aW1lc3RhbXAiOjE1NjY0NzYz
NDcwMDAsIm1lc3NhZ2UiOiJUZXN0IGV2ZW50IDMifSx7ImlkIjoiMzQ5MzM1ODk4NzM5OTQzNDEw
NTM0Nzg4MzI5NDE2NjQ3MjE2Nzg4MjY4Mzc1MzAzNzkyMjMwNDEiLCJ0aW1lc3RhbXAiOjE1NjY0
NzYzNDcwMDEsIm1lc3NhZ2UiOiJUZXN0IGV2ZW50IDQifSx7ImlkIjoiMzQ5MzM1ODk4NzQwMTY2
NDE3OTg2NzczNjM1NjQ4MDYyNTczOTcwOTk0ODU4OTE4ODUyMDM0NTgiLCJ0aW1lc3RhbXAiOjE1
NjY0NzYzNDcwMDIsIm1lc3NhZ2UiOiJUaGlzIG1lc3NhZ2UgYWxzbyBjb250YWlucyBhbiBFcnJv
ciJ9XX0=`),
					PartitionKey:         "abc123",
					SequenceNumber:       "12345",
					KinesisSchemaVersion: "1.0",
					EncryptionType:       "test",
				},
			},
		},
	}

	events, err := CloudwatchKinesisEvent(request, true, false)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(events))

	envelopeFields := common.MapStr{
		"event_id":                "1234",
		"event_name":              "connect",
		"event_source":            "web",
		"event_source_arn":        "arn:aws:iam::00000000:role/functionbeat",
		"event_version":           "1.0",
		"aws_region":              "us-east-1",
		"kinesis_partition_key":   "abc123",
		"kinesis_schema_version":  "1.0",
		"kinesis_sequence_number": "12345",
		"kinesis_encryption_type": "test",
	}

	expectedInnerFields := []common.MapStr{
		common.MapStr{
			"id":           "34933589873972040308280302318523185960554189168873242624",
			"log_group":    "testem",
			"log_stream":   "folyomany",
			"owner":        "079709761152",
			"message":      "Test event 3",
			"message_type": "DATA_MESSAGE",
			"subscription_filters": []string{
				"Minden",
			},
		},
		common.MapStr{
			"id":           "34933589873994341053478832941664721678826837530379223041",
			"log_group":    "testem",
			"log_stream":   "folyomany",
			"owner":        "079709761152",
			"message":      "Test event 4",
			"message_type": "DATA_MESSAGE",
			"subscription_filters": []string{
				"Minden",
			},
		},
		common.MapStr{
			"id":           "34933589874016641798677363564806257397099485891885203458",
			"log_group":    "testem",
			"log_stream":   "folyomany",
			"owner":        "079709761152",
			"message":      "This message also contains an Error",
			"message_type": "DATA_MESSAGE",
			"subscription_filters": []string{
				"Minden",
			},
		},
	}

	for i, expectedFields := range expectedInnerFields {
		expectedFields.Update(envelopeFields)
		assert.Equal(t, expectedFields, events[i].Fields)
	}
}
