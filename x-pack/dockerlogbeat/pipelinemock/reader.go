// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemock

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

// CreateTestInput creates a mocked ReadCloser for the pipelineReader
func CreateTestInput(t *testing.T) io.ReadCloser {
	//setup
	exampleStruct := &logdriver.LogEntry{
		Source:   "Test",
		TimeNano: 0,
		Line:     []byte("This is a log line"),
		Partial:  false,
		PartialLogMetadata: &logdriver.PartialLogEntryMetadata{
			Last:    false,
			Id:      "",
			Ordinal: 0,
		},
	}

	rawBytes, err := proto.Marshal(exampleStruct)
	assert.NoError(t, err)

	sizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBytes, uint32(len(rawBytes)))
	rawBytes = append(sizeBytes, rawBytes...)
	rc := ioutil.NopCloser(bytes.NewReader(rawBytes))
	return rc
}
