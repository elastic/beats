// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/cloudwatchlogsiface"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestGetStartPosition(t *testing.T) {
	currentTime := time.Date(2020, time.June, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		title             string
		startPosition     string
		prevEndTime       int64
		scanFrequency     time.Duration
		latency           time.Duration
		expectedStartTime int64
		expectedEndTime   int64
	}{
		{
			"startPosition=beginning",
			"beginning",
			int64(0),
			30 * time.Second,
			0,
			int64(0),
			int64(1590969600000),
		},
		{
			"startPosition=end",
			"end",
			int64(0),
			30 * time.Second,
			0,
			int64(1590969570000),
			int64(1590969600000),
		},
		{
			"startPosition=typo",
			"typo",
			int64(0),
			30 * time.Second,
			0,
			int64(0),
			int64(0),
		},
		{
			"startPosition=beginning with prevEndTime",
			"beginning",
			int64(1590000000000),
			30 * time.Second,
			0,
			int64(1590000000000),
			int64(1590969600000),
		},
		{
			"startPosition=end with prevEndTime",
			"end",
			int64(1590000000000),
			30 * time.Second,
			0,
			int64(1590000000000),
			int64(1590969600000),
		},
		{
			"startPosition=beginning with latency",
			"beginning",
			int64(0),
			30 * time.Second,
			10 * time.Minute,
			int64(0),
			int64(1590969000000),
		},
		{
			"startPosition=beginning with prevEndTime and latency",
			"beginning",
			int64(1590000000000),
			30 * time.Second,
			10 * time.Minute,
			int64(1590000000000),
			int64(1590969000000),
		},
		{
			"startPosition=end with latency",
			"end",
			int64(0),
			30 * time.Second,
			10 * time.Minute,
			int64(1590968970000),
			int64(1590969000000),
		},
		{
			"startPosition=end with prevEndTime and latency",
			"end",
			int64(1590000000000),
			30 * time.Second,
			10 * time.Minute,
			int64(1590000000000),
			int64(1590969000000),
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			startTime, endTime := getStartPosition(c.startPosition, currentTime, c.prevEndTime, c.scanFrequency, c.latency)
			assert.Equal(t, c.expectedStartTime, startTime)
			assert.Equal(t, c.expectedEndTime, endTime)
		})
	}
}

func TestCreateEvent(t *testing.T) {
	logEvent := cloudwatchlogs.FilteredLogEvent{
		EventId:       awssdk.String("id-1"),
		IngestionTime: awssdk.Int64(1590000000000),
		LogStreamName: awssdk.String("logStreamName1"),
		Message:       awssdk.String("test-message-1"),
		Timestamp:     awssdk.Int64(1600000000000),
	}

	expectedEventFields := common.MapStr{
		"message": "test-message-1",
		"event": common.MapStr{
			"id": *logEvent.EventId,
		},
		"log.file.path": "logGroup1" + "/" + *logEvent.LogStreamName,
		"awscloudwatch": common.MapStr{
			"log_group":      "logGroup1",
			"log_stream":     *logEvent.LogStreamName,
			"ingestion_time": time.Unix(*logEvent.IngestionTime/1000, 0),
		},
		"cloud": common.MapStr{
			"provider": "aws",
			"region":   "us-east-1",
		},
	}
	event := createEvent(logEvent, "logGroup1", "us-east-1")
	event.Fields.Delete("event.ingested")
	assert.Equal(t, expectedEventFields, event.Fields)
}

func TestParseARN(t *testing.T) {
	logGroup, regionName, err := parseARN("arn:aws:logs:us-east-1:428152502467:log-group:test:*")
	assert.Equal(t, "test", logGroup)
	assert.Equal(t, "us-east-1", regionName)
	assert.NoError(t, err)
}
