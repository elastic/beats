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

//go:build integration

package integration

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// counts number of lines in the given file and  asserts if it matches expected count
func CountLinesInFile(t *testing.T, path string, count int) {
	t.Helper()
	var lines []byte
	var err error
	require.Eventuallyf(t, func() bool {
		// ensure all log lines are ingested
		lines, err = os.ReadFile(path)
		if err != nil {
			t.Logf("error reading file %v", err)
			return false
		}
		lines := strings.Split(string(lines), "\n")
		// we subtract number of lines by 1 because the last line in output file contains an extra \n
		return len(lines)-1 == count
	}, 2*time.Minute, 10*time.Second, "expected lines: %d, got lines: %d", count, lines)

}
