// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipereader

import (
	"testing"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/x-pack/dockerlogbeat/pipelinemock"
)

func TestPipeReader(t *testing.T) {

	TestLine := "This is a log line"
	reader := pipelinemock.CreateTestInputFromLine(t, TestLine)

	// actual test
	pipeRead, err := NewReaderFromReadCloser(reader)
	assert.NoError(t, err)
	var outLog logdriver.LogEntry
	err = pipeRead.ReadMessage(&outLog)
	assert.NoError(t, err)

	assert.Equal(t, TestLine, string(outLog.Line))

}
