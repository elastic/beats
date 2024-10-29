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
		"log": mapstr.M{
			"file": mapstr.M{
				"path": "logGroup1" + "/" + *logEvent.LogStreamName,
			},
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

func Test_FromConfig(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config
		awsCfg       awssdk.Config
		expectGroups []string
		expectRegion string
		isError      bool
	}{
		{
			name: "Valid log group ARN",
			cfg: config{
				LogGroupARN: "arn:aws:logs:us-east-1:123456789012:myLogs",
			},
			awsCfg: awssdk.Config{
				Region: "us-east-1",
			},
			expectGroups: []string{"arn:aws:logs:us-east-1:123456789012:myLogs"},
			expectRegion: "us-east-1",
			isError:      false,
		},
		{
			name: "Invalid ARN results in an error",
			cfg: config{
				LogGroupARN: "invalidARN",
			},
			awsCfg: awssdk.Config{
				Region: "us-east-1",
			},
			expectRegion: "",
			isError:      true,
		},
		{
			name: "Valid log group ARN but empty region cause error",
			cfg: config{
				LogGroupARN: "arn:aws:logs::123456789012:otherLogs",
			},
			awsCfg: awssdk.Config{
				Region: "us-east-1",
			},
			expectRegion: "",
			isError:      true,
		},
		{
			name: "ARN suffix trimming to match logGroupIdentifier requirement",
			cfg: config{
				LogGroupARN: "arn:aws:logs:us-east-1:123456789012:log-group:/aws/kinesisfirehose/ProjectA:*",
			},
			awsCfg: awssdk.Config{
				Region: "us-east-1",
			},
			expectGroups: []string{"arn:aws:logs:us-east-1:123456789012:log-group:/aws/kinesisfirehose/ProjectA"},
			expectRegion: "us-east-1",
			isError:      false,
		},
		{
			name: "LogGroupName only",
			cfg: config{
				LogGroupName: "myLogGroup",
			},
			awsCfg: awssdk.Config{
				Region: "us-east-1",
			},
			expectGroups: []string{"myLogGroup"},
			expectRegion: "us-east-1",
			isError:      false,
		},
		{
			name: "LogGroupName and region override",
			cfg: config{
				LogGroupName: "myLogGroup",
				RegionName:   "sa-east-1",
			},
			awsCfg: awssdk.Config{
				Region: "us-east-1",
			},
			expectGroups: []string{"myLogGroup"},
			expectRegion: "sa-east-1",
			isError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups, region, err := fromConfig(tt.cfg, tt.awsCfg)
			if tt.isError {
				assert.Error(t, err)
			}

			assert.Equal(t, tt.expectGroups, groups)
			assert.Equal(t, tt.expectRegion, region)
		})
	}
}
