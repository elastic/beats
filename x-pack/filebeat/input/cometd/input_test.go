// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cometd

import (
	"testing"

	"github.com/elastic/beats/v7/filebeat/input/inputtest"
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestNewInputDone(t *testing.T) {
	config := common.MapStr{
		"channel_name": "some-channel",
	}
	inputtest.AssertNotStartedInputCanBeDone(t, NewInput, &config)
}
