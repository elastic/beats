// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/inputtest"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestNewInputDone(t *testing.T) {
	config := common.MapStr{
		"channel_name":              "cometd-channel",
		"auth.oauth2.client.id":     "DEMOCLIENTID",
		"auth.oauth2.client.secret": "DEMOCLIENTSECRET",
		"auth.oauth2.user":          "salesforce_user",
		"auth.oauth2.password":      "P@$$w0₹D",
		"auth.oauth2.token_url":     "https://login.salesforce.com/services/oauth2/token",
	}
	inputtest.AssertNotStartedInputCanBeDone(t, NewInput, &config)
}

func TestMakeEventFailure(t *testing.T) {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			"event": common.MapStr{
				"id":      "DEMOID",
				"created": time.Now().UTC(),
			},
			"message": "DEMOBODYFAIL",
		},
		Private: "DEMOBODYFAIL",
	}
	assert.NotEqual(t, event, makeEvent("DEMOID", "DEMOBODY"))
}
