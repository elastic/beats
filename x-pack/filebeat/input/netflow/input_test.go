// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package netflow

import (
	"testing"

	"github.com/elastic/beats/v7/filebeat/input/inputtest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestNewInputDone(t *testing.T) {
	config := mapstr.M{}
	inputtest.AssertNotStartedInputCanBeDone(t, NewInput, &config)
}
