// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/functionbeat/provider"
)

func TestKinesis(t *testing.T) {
	cfg := common.MustNewConfigFrom(map[string]interface{}{
		"name": "foobar",
	})

	t.Run("when publish is succesful", func(t *testing.T) {
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
