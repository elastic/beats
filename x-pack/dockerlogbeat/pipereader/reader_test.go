// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipereader

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/docker/docker/api/types/plugins/logdriver"
)

func TestPipeReader(t *testing.T) {

	rawBytes := pipeineMock.CreateTestInput(t)

	// actual test
	pipeRead, err := NewReaderFromReadCloser(rawBytes)
	assert.NoError(t, err)
	var outLog logdriver.LogEntry
	err = pipeRead.ReadMessage(&outLog)
	assert.NoError(t, err)

	assert.Equal(t, "This is a log line", string(outLog.Line))

}
