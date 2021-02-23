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
	"os"
	"runtime"
	"testing"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
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

// test_close_removed from test_harvester.py
func TestFilestreamCloseRemoved(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                                []string{env.abspath(testlogName) + "*"},
		"prospector.scanner.check_interval":    "24h",
		"close.on_state_change.check_interval": "1ms",
		"close.on_state_change.removed":        "true",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first log line\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	// first event has made it successfully
	env.waitUntilEventCount(1)
	// check registry
	env.requireOffsetInRegistry(testlogName, len(testlines))

	fi, err := os.Stat(env.abspath(testlogName))
	if err != nil {
		t.Fatalf("cannot stat file: %+v", err)
	}

	env.mustRemoveFile(testlogName)

	// the second log line will not be picked up as scan_interval is set to one day.
	env.mustWriteLinesToFile(testlogName, []byte("first line\nsecond log line\n"))

	// new two events arrived
	env.waitUntilEventCount(1)

	cancelInput()
	env.waitUntilInputStops()

	identifier, _ := newINodeDeviceIdentifier(nil)
	src := identifier.GetSource(loginp.FSEvent{Info: fi, Op: loginp.OpCreate, NewPath: env.abspath(testlogName)})
	env.requireOffsetInRegistryByID(src.Name(), len(testlines))
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

// test_empty_lines from test_harvester.py
func TestFilestreamEmptyLine(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("first log line\nnext is an empty line\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	env.waitUntilEventCount(2)
	env.requireOffsetInRegistry(testlogName, len(testlines))

	moreTestlines := []byte("\nafter an empty line\n")
	env.mustAppendLinesToFile(testlogName, moreTestlines)

	env.waitUntilEventCount(3)

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, len(testlines)+len(moreTestlines))
}

// test_empty_lines_only from test_harvester.py
// This test differs from the original because in filestream
// input offset is no longer persisted when the line is empty.
func TestFilestreamEmptyLinesOnly(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":                             []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval": "1ms",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testlines := []byte("\n\n\n")
	env.mustWriteLinesToFile(testlogName, testlines)

	cancelInput()
	env.waitUntilInputStops()

	env.requireNoEntryInRegistry(testlogName)
}

// test_exceed_buffer from test_harvester.py
func TestFilestreamExceedBuffer(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"paths":       []string{env.abspath(testlogName)},
		"buffer_size": 10,
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	testline := []byte("a line longer than size allowed in buffer_size\n")
	expectedOffset := len(testline)
	env.mustWriteLinesToFile(testlogName, testline)

	// event arrives to the output in full
	env.waitUntilEventCount(1)
	env.requireEventsReceived([]string{string(testline[:len(testline)-1])})

	cancelInput()
	env.waitUntilInputStops()

	env.requireOffsetInRegistry(testlogName, expectedOffset)
}
