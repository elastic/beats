// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// TestGetBeatInfoInitializesPaths guards against a regression where
// beat.Info.Paths was left nil for dockerlogbeat. Output factories (e.g.
// elasticsearch) read queue paths from beat.Info.Paths, so a nil value
// causes disk queue creation to fail with "got nil paths" and the
// pipeline silently falls back to the memory queue.
func TestGetBeatInfoInitializesPaths(t *testing.T) {
	info, err := getBeatInfo(ContainerOutputConfig{BeatName: "dockerlogbeat-test"}, "localhost", logptest.NewTestingLogger(t, "test"))
	assert.NoError(t, err)
	assert.NotNil(t, info.Paths, "beat.Info.Paths must be initialized so queue.disk output config can create a disk queue")
}
