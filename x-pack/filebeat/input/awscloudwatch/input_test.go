// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestCreateEvent(t *testing.T) {
	logEvent := &types.FilteredLogEvent{
		EventId:       awssdk.String("id-1"),
		IngestionTime: awssdk.Int64(1590000000000),
		LogStreamName: awssdk.String("logStreamName1"),
		Message:       awssdk.String("test-message-1"),
		Timestamp:     awssdk.Int64(1600000000000),
	}

	expectedEventFields := mapstr.M{
		"message": "test-message-1",
		"event": mapstr.M{
			"id": *logEvent.EventId,
		},
		"log.file.path": "logGroup1" + "/" + *logEvent.LogStreamName,
		"awscloudwatch": mapstr.M{
			"log_group":      "logGroup1",
			"log_stream":     *logEvent.LogStreamName,
			"ingestion_time": time.Unix(*logEvent.IngestionTime/1000, 0),
		},
		"aws.cloudwatch": mapstr.M{
			"log_group":      "logGroup1",
			"log_stream":     *logEvent.LogStreamName,
			"ingestion_time": time.Unix(*logEvent.IngestionTime/1000, 0),
		},
		"cloud": mapstr.M{
			"provider": "aws",
			"region":   "us-east-1",
		},
	}
	event := createEvent(*logEvent, "logGroup1", "us-east-1")
	err := event.Fields.Delete("event.ingested")
	assert.NoError(t, err)
	assert.Equal(t, expectedEventFields, event.Fields)
}

func TestParseARN(t *testing.T) {
	logGroup, regionName, err := parseARN("arn:aws:logs:us-east-1:428152502467:log-group:test:*")
	assert.Equal(t, "test", logGroup)
	assert.Equal(t, "us-east-1", regionName)
	assert.NoError(t, err)
}
