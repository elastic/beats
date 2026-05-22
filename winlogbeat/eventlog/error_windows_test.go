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

//go:build windows

package eventlog

import (
	"testing"

	win "github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
	"github.com/stretchr/testify/assert"
)

func TestIsRecoverable(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		isFile bool
		want   bool
	}{
		{"RPC_S_UNKNOWN_IF is recoverable", win.RPC_S_UNKNOWN_IF, false, true},
		{"RPC_S_SERVER_UNAVAILABLE recoverable", win.RPC_S_SERVER_UNAVAILABLE, false, true},
		{"RPC_S_CALL_CANCELLED recoverable", win.RPC_S_CALL_CANCELLED, false, true},
		{"ERROR_INVALID_HANDLE recoverable", win.ERROR_INVALID_HANDLE, false, true},
		{"nil is not recoverable", nil, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsRecoverable(tc.err, tc.isFile))
		})
	}
}
