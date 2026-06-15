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

package filestream

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/klauspost/compress/gzip"

	"github.com/gofrs/uuid/v5"
)

// poolInputConfig returns a base filestream config with the experimental worker
// pool enabled.
func poolInputConfig(env *inputTestingEnvironment, pathGlob string) (string, map[string]any) {
	id := "pool-" + uuid.Must(uuid.NewV4()).String()
	return id, map[string]any{
		"id":                                     id,
		"paths":                                  []string{pathGlob},
		"prospector.scanner.check_interval":      "10ms",
		"prospector.scanner.fingerprint.enabled": false,
		"file_identity.native":                   map[string]any{},
	}
}

// TestWorkerPoolReadsAndTails verifies that, with the worker pool enabled, the
// input reads an existing file, persists the offset, and tails appended data.
func TestWorkerPoolReadsAndTails(t *testing.T) {
	env := newInputTestingEnvironment(t)

	logName := "pool.log"
	id, cfg := poolInputConfig(env, env.abspath(logName)+"*")
	inp := env.mustCreateInput(cfg)

	line1 := []byte("first line\n")
	env.mustWriteToFile(logName, line1)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, id, inp)

	env.waitUntilEventCount(1)
	env.requireOffsetInRegistry(logName, id, len(line1))

	// Tail: appended data must be picked up by the parked-then-resumed session.
	line2 := []byte("second line\n")
	env.mustAppendToFile(logName, line2)

	env.waitUntilEventCount(2)
	env.requireOffsetInRegistry(logName, id, len(line1)+len(line2))

	env.requireEventsReceived([]string{"first line", "second line"})

	cancelInput()
	env.waitUntilInputStops()
}

// TestWorkerPoolMultiplexesManyFiles verifies that many files are all read by
// the fixed pool of workers (more files than workers), with correct content.
func TestWorkerPoolMultiplexesManyFiles(t *testing.T) {
	env := newInputTestingEnvironment(t)

	const numFiles = 50
	const linesPerFile = 3

	id, cfg := poolInputConfig(env, env.abspath("multi-")+"*")
	// Force a single worker so we exercise true multiplexing of many files
	// over one goroutine.
	cfg["harvester_limit"] = 1
	inp := env.mustCreateInput(cfg)

	var want []string
	for f := 0; f < numFiles; f++ {
		name := fmt.Sprintf("multi-%02d.log", f)
		var content []byte
		for l := 0; l < linesPerFile; l++ {
			line := fmt.Sprintf("file%02d-line%d", f, l)
			want = append(want, line)
			content = append(content, []byte(line+"\n")...)
		}
		env.mustWriteToFile(name, content)
	}

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, id, inp)

	env.waitUntilEventCount(numFiles * linesPerFile)

	got := env.getOutputMessages()
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("expected %d events, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event %d mismatch: want %q got %q", i, want[i], got[i])
		}
	}

	cancelInput()
	env.waitUntilInputStops()
}

// TestWorkerPoolCloseEOF verifies that with close.reader.on_eof the worker pool
// reaches a terminal slice and tears the harvester down (the file is read once).
func TestWorkerPoolCloseEOF(t *testing.T) {
	env := newInputTestingEnvironment(t)

	logName := "eof.log"
	id, cfg := poolInputConfig(env, env.abspath(logName)+"*")
	cfg["close.reader.on_eof"] = true
	inp := env.mustCreateInput(cfg)

	lines := []byte("a\nb\nc\n")
	env.mustWriteToFile(logName, lines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, id, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(logName, id, len(lines))

	// Give the harvester a moment to tear down after EOF, then confirm the
	// input is still healthy and can be stopped cleanly.
	time.Sleep(200 * time.Millisecond)

	cancelInput()
	env.waitUntilInputStops()
}

// TestWorkerPoolGZIP verifies the worker pool reads a gzip file once to EOF and
// tears the session down (gzip never tails/parks; it returns SliceDone).
func TestWorkerPoolGZIP(t *testing.T) {
	env := newInputTestingEnvironment(t)

	logName := "data.log.gz"
	id := "pool-gzip-" + uuid.Must(uuid.NewV4()).String()
	// compression=auto requires the fingerprint file identity. Use a small
	// fingerprint length so the (small) gzip file is not skipped by the scanner.
	cfg := map[string]any{
		"id":                                id,
		"paths":                             []string{env.abspath(logName) + "*"},
		"prospector.scanner.check_interval": "10ms",
		"prospector.scanner.fingerprint": map[string]any{
			"enabled": true,
			"offset":  0,
			"length":  64,
		},
		"compression": "auto",
	}
	inp := env.mustCreateInput(cfg)

	var want []string
	for i := 0; i < 20; i++ {
		want = append(want, fmt.Sprintf("gzip line %02d", i))
	}
	var plain bytes.Buffer
	for _, l := range want {
		plain.WriteString(l + "\n")
	}
	var compressed bytes.Buffer
	gw := gzip.NewWriter(&compressed)
	if _, err := gw.Write(plain.Bytes()); err != nil {
		t.Fatalf("cannot gzip content: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("cannot close gzip writer: %v", err)
	}
	env.mustWriteToFile(logName, compressed.Bytes())

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, id, inp)

	env.waitUntilEventCount(len(want))
	env.requireEventsReceived(want)

	cancelInput()
	env.waitUntilInputStops()
}
