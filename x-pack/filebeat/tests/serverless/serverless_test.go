// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && serverless

package serverless

import (
	"testing"

	"github.com/elastic/beats/v7/testing/go-serverless/beatsuite"
)

// TestFilebeatServerless runs Filebeat's setup and export commands, and a
// short run, against a serverless project. It needs EC_API_KEY and BEAT_HOME,
// see .buildkite/scripts/serverless_tests.sh.
func TestFilebeatServerless(t *testing.T) {
	beatsuite.Run(t, beatsuite.Config{BeatName: "filebeat"})
}
