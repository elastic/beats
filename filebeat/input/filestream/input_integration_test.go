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

// +build integration

package filestream

import (
	"context"
	"runtime"
	"testing"
)

// test_close_renamed from test_harvester.py
func TestFilestreamCloseRenamed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("renaming files while Filebeat is running is not supported on Windows")
	}

	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                                []string{env.abspath(testlogName) + "*"},
		"prospector.scanner.check_interval":    "1ms",
		"close.on_state_change.check_interval": "1ms",
		"close.on_state_change.renamed":        "true",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first log line\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	// first event has made it successfully
	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, len(testlines))

	testlogNameRotated := "test.log.rotated"
	env.mustRenameFile(testlogName, testlogNameRotated)

	newerTestlines := []byte("new first log line\nnew second log line\n")
	env.mustWriteLinesToFile(testlogName, newerTestlines)

	// new two events arrived
	env.waitUntilEventCount(3)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogNameRotated, len(testlines))
	env.requireOffsetInRegistry(testlogName, len(newerTestlines))
}

// test_close_eof from test_harvester.py
func TestFilestreamCloseEOF(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "24h",
		"close.reader.on_eof":               "true",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first log line\n")
	expectedOffset := len(testlines)
	env.mustWriteLinesToFile(testlogName, testlines)

	// first event has made it successfully
	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(testlogName, expectedOffset)

	// the second log line will not be picked up as scan_interval is set to one day.
	env.mustWriteLinesToFile(testlogName, []byte("first line\nsecond log line\n"))

	// only one event is read
	env.waitUntilEventCount(1)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, expectedOffset)
}
