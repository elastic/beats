// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemock

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

// CreateTestInputFromLine returns a ReadCloser based on an input string
func CreateTestInputFromLine(t *testing.T, line string) io.ReadCloser {
	exampleStruct := &logdriver.LogEntry{
		Source:   "Test",
		TimeNano: 0,
		Line:     []byte(line),
		Partial:  false,
		PartialLogMetadata: &logdriver.PartialLogEntryMetadata{
			Last:    false,
			Id:      "",
			Ordinal: 0,
		},
	}

	writer := new(bytes.Buffer)

	encodeLog(t, writer, exampleStruct)
	return io.NopCloser(writer)
}

func encodeLog(t *testing.T, out io.Writer, entry *logdriver.LogEntry) {
	rawBytes, err := proto.Marshal(entry)
	require.NoError(t, err)

	sizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBytes, uint32(len(rawBytes)))

	_, err = out.Write(sizeBytes)
	require.NoError(t, err)
	_, err = out.Write(rawBytes)
	require.NoError(t, err)
}
