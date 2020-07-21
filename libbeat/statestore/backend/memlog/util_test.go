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

type mockErrorWriter struct {
	reportedError bool
}

func (mew *mockErrorWriter) Write(data []byte) (n int, err error) {
	if !mew.reportedError {
		mew.reportedError = true
		return 0, syscall.EAGAIN
	}
	return len(data), nil
}

func TestEnsureWriter_RetriableError(t *testing.T) {
	errorWriter := &mockErrorWriter{}
	writer := &ensureWriter{errorWriter}
	written, err := writer.Write([]byte{1, 2, 3})
	if err != nil {
		t.Fatalf("EnsureWriter shouldn't propagate retriable errors")
	}
	if written != 3 {
		t.Fatalf("Expected 3 bytes written, got %d", written)
	}
}
