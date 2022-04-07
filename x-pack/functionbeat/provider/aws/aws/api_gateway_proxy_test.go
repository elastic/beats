// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/x-pack/functionbeat/function/provider"
)

func TestAPIGatewayProxy(t *testing.T) {
	cfg := common.MustNewConfigFrom(map[string]interface{}{
		"name": "foobar",
	})

	t.Run("when publish is succesful", func(t *testing.T) {
		t.SkipNow()
		client := &arrayBackedClient{}
		s, err := NewAPIGatewayProxy(&provider.DefaultProvider{}, cfg)
		if !assert.NoError(t, err) {
			return
		}

		c, _ := s.(*APIGatewayProxy)
		handler := c.createHandler(client)
		res, err := handler(generateAPIGatewayProxyEvent())
		assert.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		ty, _ := res.Headers["Content-Type"]
		assert.Equal(t, "application/json", ty)

		message, err := unserializeResponse(res.Body)
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "1234", message.RequestID)
		assert.Equal(t, "event received successfully.", message.Message)
		assert.Equal(t, http.StatusOK, message.Status)
	})

	t.Run("when publish is not succesful", func(t *testing.T) {
		e := errors.New("something bad")
		client := &arrayBackedClient{err: e}

		s, err := NewAPIGatewayProxy(&provider.DefaultProvider{}, cfg)
		if !assert.NoError(t, err) {
			return
		}

		c, _ := s.(*APIGatewayProxy)
		res, err := c.createHandler(client)(generateAPIGatewayProxyEvent())
		assert.Equal(t, e, err)
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		ty, _ := res.Headers["Content-Type"]
		assert.Equal(t, "application/json", ty)

		message, err := unserializeResponse(res.Body)
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "1234", message.RequestID)
		assert.Equal(t, "an error occurred when sending the event.", message.Message)
		assert.Equal(t, http.StatusInternalServerError, message.Status)
	})
}

func generateAPIGatewayProxyEvent() events.APIGatewayProxyRequest {
	return events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "1234",
		},
	}
}

func unserializeResponse(raw string) (*message, error) {
	message := &message{}
	if err := json.Unmarshal([]byte(raw), message); err != nil {
		return nil, err
	}
	return message, nil
}
