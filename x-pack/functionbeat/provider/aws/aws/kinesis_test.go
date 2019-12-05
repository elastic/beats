// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/functionbeat/function/provider"
)

func TestKinesis(t *testing.T) {
	cfg := common.MustNewConfigFrom(map[string]interface{}{
		"name": "foobar",
		"triggers": []map[string]interface{}{
			map[string]interface{}{
				"event_source_arn": "abc123",
			},
		},
	})

	t.Run("when publish is successful", func(t *testing.T) {
		client := &arrayBackedClient{}
		k, err := NewKinesis(&provider.DefaultProvider{}, cfg)
		if !assert.NoError(t, err) {
			return
		}

		c, _ := k.(*Kinesis)
		handler := c.createHandler(client)
		err = handler(generateKinesisEvent())
		assert.NoError(t, err)
	})

	t.Run("when publish is not succesful", func(t *testing.T) {
		e := errors.New("something bad")
		client := &arrayBackedClient{err: e}

		k, err := NewKinesis(&provider.DefaultProvider{}, cfg)
		if !assert.NoError(t, err) {
			return
		}

		c, _ := k.(*Kinesis)
		handler := c.createHandler(client)
		err = handler(generateKinesisEvent())
		assert.Equal(t, e, err)
	})

	t.Run("test config validation", testKinesisConfig)
	t.Run("test starting position", testStartingPosition)
}

func generateKinesisEvent() events.KinesisEvent {
	return events.KinesisEvent{
		Records: []events.KinesisEventRecord{
			events.KinesisEventRecord{
				AwsRegion:      "east-1",
				EventID:        "1234",
				EventName:      "connect",
				EventSource:    "web",
				EventSourceArn: "arn:aws:iam::00000000:role/functionbeat",
				Kinesis: events.KinesisRecord{
					Data:                 []byte("hello world"),
					PartitionKey:         "abc123",
					SequenceNumber:       "12345",
					KinesisSchemaVersion: "v1",
				},
			},
		},
	}
}

func testKinesisConfig(t *testing.T) {
	tests := map[string]struct {
		valid     bool
		rawConfig map[string]interface{}
		expected  *KinesisConfig
	}{
		"minimal valid configuration": {
			valid: true,
			rawConfig: map[string]interface{}{
				"name":        "mysuperfunctionname",
				"description": "mylong description",
				"triggers": []map[string]interface{}{
					map[string]interface{}{
						"event_source_arn": "mycustomarn",
					},
				},
			},
		},
		"missing event triggers": {
			valid: false,
			rawConfig: map[string]interface{}{
				"name":        "mysuperfunctionname",
				"description": "mylong description",
			},
		},
		"empty or missing event source arn": {
			valid: false,
			rawConfig: map[string]interface{}{
				"name":        "mysuperfunctionname",
				"description": "mylong description",
				"triggers": []map[string]interface{}{
					map[string]interface{}{
						"event_source_arn": "",
					},
				},
			},
		},
		"test upper bound batch size limit": {
			valid: false,
			rawConfig: map[string]interface{}{
				"name":        "mysuperfunctionname",
				"description": "mylong description",
				"triggers": []map[string]interface{}{
					map[string]interface{}{
						"event_source_arn": "abc123",
						"batch_size":       20000,
					},
				},
			},
		},
		"test lower bound batch size limit": {
			valid: false,
			rawConfig: map[string]interface{}{
				"name":        "mysuperfunctionname",
				"description": "mylong description",
				"triggers": []map[string]interface{}{
					map[string]interface{}{
						"event_source_arn": "abc123",
						"batch_size":       10,
					},
				},
			},
		},
		"test default values": {
			valid: true,
			rawConfig: map[string]interface{}{
				"name":        "mysuperfunctionname",
				"description": "mylongdescription",
				"triggers": []map[string]interface{}{
					map[string]interface{}{
						"event_source_arn": "abc123",
					},
				},
			},
			expected: &KinesisConfig{
				Name:         "mysuperfunctionname",
				Description:  "mylongdescription",
				LambdaConfig: DefaultLambdaConfig,
				Triggers: []*KinesisTriggerConfig{
					&KinesisTriggerConfig{
						EventSourceArn:   "abc123",
						BatchSize:        100,
						StartingPosition: trimHorizonPos,
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := common.MustNewConfigFrom(test.rawConfig)
			config := &KinesisConfig{LambdaConfig: DefaultLambdaConfig}
			err := cfg.Unpack(config)
			if !assert.Equal(t, test.valid, err == nil, fmt.Sprintf("error: %+v", err)) {
				return
			}

			if test.expected != nil {
				assert.Equal(t, test.expected, config)
			}
		})
	}
}

func testStartingPosition(t *testing.T) {
	// NOTE(ph) when adding support for at_timestamp we also need to make sure the cloudformation
	// template send the timestamp.
	// doc: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-eventsourcemapping.html
	t.Run("AT_TIMESTAMP is not supported yet", func(t *testing.T) {
		var s startingPosition
		err := s.Unpack("at_timestamp")
		assert.Error(t, err)
	})
}
