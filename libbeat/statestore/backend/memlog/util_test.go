// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package memlog

import (
	"syscall"
	"testing"
)

// A mock Writer implementation that always returns a configurable
// error on the first write call, to test error handling in ensureWriter.
type mockErrorWriter struct {
	errorType     error
	reportedError bool
}

func (mew *mockErrorWriter) Write(data []byte) (n int, err error) {
	if !mew.reportedError {
		mew.reportedError = true
		return 0, mew.errorType
	}
	return len(data), nil
}

func TestEnsureWriter_RetriableError(t *testing.T) {
	// EAGAIN is retriable, ensureWriter.Write should succeed.
	errorWriter := &mockErrorWriter{errorType: syscall.EAGAIN}
	bytes := []byte{1, 2, 3}
	writer := &ensureWriter{errorWriter}
	written, err := writer.Write(bytes)
	if err != nil {
		t.Fatalf("ensureWriter shouldn't propagate retriable errors")
	}
	if written != len(bytes) {
		t.Fatalf("Expected %d bytes written, got %d", len(bytes), written)
	}
}

func TestEnsureWriter_NonRetriableError(t *testing.T) {
	// EINVAL is not retriable, ensureWriter.Write should return an error.
	errorWriter := &mockErrorWriter{errorType: syscall.EINVAL}
	bytes := []byte{1, 2, 3}
	writer := &ensureWriter{errorWriter}
	written, err := writer.Write(bytes)
	if err != syscall.EINVAL {
		t.Fatalf("ensureWriter should propagate nonretriable errors")
	}
	if written != 0 {
		t.Fatalf("Expected 0 bytes written, got %d", written)
	}
}

func TestEnsureWriter_NoError(t *testing.T) {
	// This tests the case where the underlying writer returns with no error,
	// but without writing the full buffer.
	var bytes []byte = []byte{1, 2, 3}
	errorWriter := &mockErrorWriter{errorType: nil}
	writer := &ensureWriter{errorWriter}
	written, err := writer.Write(bytes)
	if err != nil {
		t.Fatalf("ensureWriter should only error if the underlying writer does")
	}
	if written != len(bytes) {
		t.Fatalf("Expected %d bytes written, got %d", len(bytes), written)
	}
}
