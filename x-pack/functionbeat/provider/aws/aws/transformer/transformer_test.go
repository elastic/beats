// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transformer

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/kinesis-aggregation/go/deaggregator"
	aggRecProto "github.com/awslabs/kinesis-aggregation/go/records"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
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
	assert.NoError(t, err)

	expectedEvent := beat.Event{
		Timestamp: expectedTime,
		Fields: mapstr.M{
			"event": mapstr.M{
				"kind": "event",
			},
			"cloud": mapstr.M{
				"provider": "aws",
			},
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
	t.Run("when kinesis event is successful", func(t *testing.T) {
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

		events, err := KinesisEvent(request)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(events))

		fields := mapstr.M{
			"cloud": mapstr.M{
				"provider": "aws",
				"region":   "us-east-1",
			},
			"event": mapstr.M{
				"kind": "event",
			},
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
	})

	t.Run("when kinesis event with agg is successful", func(t *testing.T) {
		rand.Seed(time.Now().UnixNano())
		min, max := 2, 20
		numRecords := rand.Intn(max-min) + min
		aggRecBytes := generateKinesisAggregateRecord(numRecords, true)

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
						Data:                 aggRecBytes,
						PartitionKey:         "ignored",
						SequenceNumber:       "12345",
						KinesisSchemaVersion: "1.0",
						EncryptionType:       "test",
					},
				},
			},
		}

		events, err := KinesisEvent(request)
		assert.NoError(t, err)
		assert.Equal(t, numRecords, len(events))

		envelopeFields := mapstr.M{
			"cloud": mapstr.M{
				"provider": "aws",
				"region":   "us-east-1",
			},
			"event": mapstr.M{
				"kind": "event",
			},
			"event_id":                "1234",
			"event_name":              "connect",
			"event_source":            "web",
			"event_source_arn":        "arn:aws:iam::00000000:role/functionbeat",
			"event_version":           "1.0",
			"aws_region":              "us-east-1",
			"kinesis_schema_version":  "1.0",
			"kinesis_sequence_number": "12345",
			"kinesis_encryption_type": "test",
		}

		var expectedInnerFields []mapstr.M
		for i := 0; i < numRecords; i++ {
			expectedInnerFields = append(expectedInnerFields, mapstr.M{
				"message":               fmt.Sprintf("%s %d", "hello world", i),
				"kinesis_partition_key": fmt.Sprintf("%s %d", "partKey", i),
			})
		}

		for i, expectedFields := range expectedInnerFields {
			expectedFields.Update(envelopeFields)
			assert.Equal(t, expectedFields, events[i].Fields)
		}
	})

	t.Run("when kinesis event with agg is not successful", func(t *testing.T) {
		aggRecBytes := generateKinesisAggregateRecord(2, false)

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
						Data:                 aggRecBytes,
						PartitionKey:         "abc123",
						SequenceNumber:       "12345",
						KinesisSchemaVersion: "1.0",
						EncryptionType:       "test",
					},
				},
			},
		}

		events, err := KinesisEvent(request)
		assert.Error(t, err)
		assert.Nil(t, events)
	})

	t.Run("when kinesis event with real example agg payload is successful", func(t *testing.T) {
		rand.Seed(time.Now().UnixNano())
		numRecords := 10
		aggRecBytes, err := base64.StdEncoding.DecodeString("84mawgoIejJKSjl6dFgaEwgAGg97ImtleSI6InZhbHVlIn0aEwgAGg9" +
			"7ImtleSI6InZhbHVlIn0aEwgAGg97ImtleSI6InZhbHVlIn0aEwgAGg97ImtleSI6InZhbHVlIn0aEwgAGg97ImtleSI6InZhbHVlIn" +
			"0aEwgAGg97ImtleSI6InZhbHVlIn0aEwgAGg97ImtleSI6InZhbHVlIn0aEwgAGg97ImtleSI6InZhbHVlIn0aEwgAGg97ImtleSI6I" +
			"nZhbHVlIn0aEwgAGg97ImtleSI6InZhbHVlIn3xj2DFMGZ0aNQC7aexsnkU")
		assert.NoError(t, err)

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
						Data:                 aggRecBytes,
						PartitionKey:         "ignored",
						SequenceNumber:       "12345",
						KinesisSchemaVersion: "1.0",
						EncryptionType:       "test",
					},
				},
			},
		}

		events, err := KinesisEvent(request)
		assert.NoError(t, err)
		assert.Equal(t, numRecords, len(events))

		envelopeFields := mapstr.M{
			"cloud": mapstr.M{
				"provider": "aws",
				"region":   "us-east-1",
			},
			"event": mapstr.M{
				"kind": "event",
			},
			"event_id":                "1234",
			"event_name":              "connect",
			"event_source":            "web",
			"event_source_arn":        "arn:aws:iam::00000000:role/functionbeat",
			"event_version":           "1.0",
			"aws_region":              "us-east-1",
			"kinesis_schema_version":  "1.0",
			"kinesis_sequence_number": "12345",
			"kinesis_encryption_type": "test",
			"kinesis_partition_key":   "z2JJ9ztX",
			"message":                 `{"key":"value"}`,
		}

		for i := 0; i < numRecords; i++ {
			assert.Equal(t, envelopeFields, events[i].Fields)
		}
	})
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
	assert.NoError(t, err)
	assert.Equal(t, 3, len(events))

	envelopeFields := mapstr.M{
		"cloud": mapstr.M{
			"provider": "aws",
			"region":   "us-east-1",
		},
		"event": mapstr.M{
			"kind": "event",
		},
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

	expectedInnerFields := []mapstr.M{
		mapstr.M{
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
		mapstr.M{
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
		mapstr.M{
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

func generateKinesisAggregateRecord(numRecords int, valid bool) []byte {
	// Heavily based on https://github.com/awslabs/kinesis-aggregation/blob/master/go/deaggregator/deaggregator_test.go
	aggRec := &aggRecProto.AggregatedRecord{}
	unquotedHeader, err := strconv.Unquote(deaggregator.KplMagicHeader)
	if err != nil {
		panic(err)
	}
	aggRecBytes := []byte(unquotedHeader)
	partKeyTable := make([]string, 0)
	for i := 0; i < numRecords; i++ {
		partKey := uint64(i)
		hashKey := uint64(i)
		r := &aggRecProto.Record{
			ExplicitHashKeyIndex: &hashKey,
			Data:                 []byte(fmt.Sprintf("%s %d", "hello world", i)),
			Tags:                 make([]*aggRecProto.Tag, 0),
		}
		// This seems to be the only way to trigger the deaggregation module to return an error when needed
		if valid {
			r.PartitionKeyIndex = &partKey
		}
		aggRec.Records = append(aggRec.Records, r)
		partKeyTable = append(partKeyTable, fmt.Sprintf("%s %d", "partKey", i))
	}

	aggRec.PartitionKeyTable = partKeyTable
	data, _ := proto.Marshal(aggRec)
	md5Hash := md5.Sum(data)
	aggRecBytes = append(aggRecBytes, data...)
	aggRecBytes = append(aggRecBytes, md5Hash[:]...)

	return aggRecBytes
}
