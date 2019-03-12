// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transformer

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

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
