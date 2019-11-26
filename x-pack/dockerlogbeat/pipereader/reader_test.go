// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipereader

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gogo/protobuf/proto"

	"github.com/docker/docker/api/types/plugins/logdriver"
)

func TestPipeReader(t *testing.T) {

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

	// actual test
	pipeRead, err := NewReaderFromReadCloser(ioutil.NopCloser(bytes.NewReader(rawBytes)))
	assert.NoError(t, err)
	var outLog logdriver.LogEntry
	err = pipeRead.ReadMessage(&outLog)
	assert.NoError(t, err)

	assert.Equal(t, "This is a log line", string(outLog.Line))

}
