// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/provider"
)

func TestS3(t *testing.T) {
	cfg := common.MustNewConfigFrom(map[string]interface{}{
		"name": "foobar",
		"triggers": []map[string]interface{}{
			map[string]interface{}{
				"event_source_arn": "abc1234",
			},
		},
	})

	t.Run("when publish is succesful", func(t *testing.T) {
		t.SkipNow()
		client := &arrayBackedClient{}
		s, err := NewS3(&provider.DefaultProvider{}, cfg)
		if !assert.NoError(t, err) {
			return
		}

		c, _ := s.(*S3)
		handler := c.createHandler(client)
		err = handler(generateS3Event())
		assert.NoError(t, err)
	})

	t.Run("when publish is not succesful", func(t *testing.T) {
		t.SkipNow()
		e := errors.New("something bad")
		client := &arrayBackedClient{err: e}

		s, err := NewS3(&provider.DefaultProvider{}, cfg)
		if !assert.NoError(t, err) {
			return
		}

		c, _ := s.(*S3)
		handler := c.createHandler(client)
		err = handler(generateS3Event())
		assert.Equal(t, e, err)
	})
}

func generateS3Event() events.S3Event {
	return events.S3Event{
		Records: []events.S3EventRecord{
			events.S3EventRecord{
				AWSRegion:   "us-east-1",
				EventName:   "createObject",
				EventSource: "aws:s3",
				S3: events.S3Entity{
					SchemaVersion:   "v1",
					ConfigurationID: "abc123",
					Bucket: events.S3Bucket{
						Name: "test-bucket",
					},
					Object: events.S3Object{
						Key: "test-key",
					},
				},
			},
		},
	}
}
