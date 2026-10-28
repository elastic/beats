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

//go:build linux

package process

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFetchPidsDeduplicates verifies that fetchPidsFromNames skips a PID it has
// already processed
func TestFetchPidsDeduplicates(t *testing.T) {
	stat, err := initTestResolver(t)
	require.NoError(t, err)

	pid := os.Getpid()
	pidStr := strconv.Itoa(pid)

	// Simulate Readdirnames returning the same PID twice.
	names := []string{pidStr, pidStr}

	procMap, plist, err := stat.fetchPidsFromNames(names)
	require.NoError(t, err)

	assert.Len(t, procMap, 1, "procMap should have exactly one entry for the duplicated PID")
	assert.Len(t, plist, 1, "plist should have exactly one entry for the duplicated PID")
	assert.Contains(t, procMap, pid)
}
