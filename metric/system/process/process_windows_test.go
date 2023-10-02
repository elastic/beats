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

package process

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

func TestGetInfoForPid_numThreads(t *testing.T) {
	const want = 42
	// On windows programs always start with more threads than the code
	// is launching. It seems to be due to it loading dll/tables in parallel
	// or due to the default thread pool. See the following for more info:
	// https://stackoverflow.com/questions/42789199/why-there-are-three-unexpected-worker-threads-when-a-win32-console-application-s/42789684#42789684
	//
	// Use the Process Explorer https://learn.microsoft.com/en-us/sysinternals/downloads/process-explorer
	// to check the number of threads allocated to a process. To see the number
	// of threads in, use you'd need to add the Threads column yourself.
	// To add it:
	//   right click on one of the column titles -> select columns -> process performance tab -> select Threads
	expected := 45

	cmd := runThreads(t)
	got, err := GetInfoForPid(
		resolve.NewTestResolver("/"), cmd.Process.Pid)
	require.NoError(t, err, "failed to GetInfoForPid")

	if !got.NumThreads.Exists() {
		bs, err := json.Marshal(got)
		if err != nil {
			t.Logf("could not marshal ProcState: %v", err)
		}
		t.Fatalf("num_thread was not collected. Collected info: %s", bs)
	}

	numThreads := got.NumThreads.ValueOr(-1)
	if expected != numThreads {
		// it might be an older Windows version or, by the time we got the num_threads,
		// the program was indeed using only what it spawned
		assert.Equalf(t, want, numThreads,
			"got %d, want %d or %d. tl;dr: on Windows process starts with more "+
				"threads than they request, see the test code for details",
			numThreads, want, expected)
	}
}
