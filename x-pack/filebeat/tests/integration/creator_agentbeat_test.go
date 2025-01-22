// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && agentbeat

package integration

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func NewFilebeat(t *testing.T) *integration.BeatProc {
	return integration.NewAgentBeat(t, "filebeat", "../../../agentbeat/agentbeat.test")
}
