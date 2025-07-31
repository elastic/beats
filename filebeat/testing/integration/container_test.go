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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/testing/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestContainerInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  10,
		PrintConfigOnFail: false,
	}

	config := `
filebeat.inputs:
- type: container
  allow_deprecated_use: true
  paths:
  - %s
output.console:
  enabled: true
filebeat.registry.flush: 0s
`

	// get current working director
	path, err := os.Getwd()
	require.NoError(t, err)

	t.Run("test container input", func(t *testing.T) {

		dockerLogPath := filepath.Join(path, "files", "logs", "docker.log")
		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, dockerLogPath),
		})

		test.ExpectJSONFields(mapstr.M{
			"message":    "Moving binaries to host...",
			"stream":     "stdout",
			"input.type": "container",
		})

		test.
			ExpectEOF(dockerLogPath).
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()

	})

	t.Run(" Test container input with CRI format", func(t *testing.T) {
		criLogPath := filepath.Join(path, "files", "logs", "cri.log")
		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, criLogPath),
		})

		test.ExpectJSONFields(mapstr.M{
			"stream":     "stdout",
			"input.type": "container",
		})

		test.
			ExpectEOF(criLogPath).
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()

	})

	t.Run(" Test container input properly updates registry offset in case of unparsable lines", func(t *testing.T) {
		dockerCorruptedPath := filepath.Join(path, "files", "logs", "docker_corrupted.log")
		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, dockerCorruptedPath),
		})

		test.ExpectJSONFields(mapstr.M{
			"message":    "Moving binaries to host...",
			"stream":     "stdout",
			"input.type": "container",
		})

		//expect parse line error
		test.ExpectOutput("Parse line error")

		test.
			ExpectEOF(dockerCorruptedPath).
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()

		registryLogFile := filepath.Join(test.GetTempDir(), "data/registry/filebeat/log.json")

		time.Sleep(1 * time.Minute)
		// bytes of healthy file are 2244 so for the corrupted one should
		// be 2244-1=2243 since we removed one character
		assertLastOffset(t, registryLogFile, 2243)

	})
}

func assertLastOffset(t *testing.T, path string, offset int) {
	t.Helper()
	entries, _ := readFilestreamRegistryLog(t, path)
	lastEntry := entries[len(entries)-1]
	if lastEntry.Offset != offset {
		t.Errorf("expecting offset %d got %d instead", offset, lastEntry.Offset)
		t.Log("last registry entries:")

		max := len(entries)
		if max > 10 {
			max = 10
		}
		for _, e := range entries[:max] {
			t.Logf("%+v\n", e)
		}

		t.FailNow()
	}
}

type registryEntry struct {
	Key      string
	Offset   int
	EOF      bool
	Filename string
	TTL      time.Duration
	Op       string
	Removed  bool
}

func readFilestreamRegistryLog(t *testing.T, path string) ([]registryEntry, map[string]string) {
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("could not open file '%s': %s", path, err)
	}

	var entries []registryEntry
	fileNameToNative := map[string]string{}
	s := bufio.NewScanner(file)

	var lastOperation string
	for s.Scan() {
		line := s.Bytes()

		e := entry{}
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatalf("could not read line '%s': %s", string(line), err)
		}

		// Skips registry log entries containing the operation ID like:
		// '{"op":"set","id":46}'
		if e.Key == "" {
			lastOperation = e.Op
			continue
		}
		// Filestream entry
		et := registryEntry{
			Key:      e.Key,
			Offset:   e.Value.Cursor.Offset,
			EOF:      e.Value.Cursor.EOF,
			TTL:      e.Value.TTL,
			Filename: e.Value.Meta.Source,
			Removed:  lastOperation == "remove",
			Op:       lastOperation,
		}

		// Handle the log input entries, they have a different format.
		if strings.HasPrefix(e.Key, "filebeat::logs") {
			et.Offset = e.Value.Offset
			et.Filename = e.Value.Source

			if lastOperation != "set" {
				continue
			}

			// Extract the native file identity so we can update the
			// expected registry accordingly
			name := filepath.Base(et.Filename)
			id := strings.Join(strings.Split(et.Key, "::")[2:], "::")
			fileNameToNative[name] = id
		}

		entries = append(entries, et)
	}

	return entries, fileNameToNative
}

type entry struct {
	Key   string `json:"k"`
	Value struct {
		// Filestream fields
		Cursor struct {
			Offset int  `json:"offset"`
			EOF    bool `json:"eof"`
		} `json:"cursor"`
		Meta struct {
			Source string `json:"source"`
		} `json:"meta"`

		// Log input fields
		Source string `json:"source"`
		Offset int    `json:"offset"`

		// Common to both inputs
		TTL time.Duration `json:"ttl"`
	} `json:"v"`

	// Keys to read the "operation"
	// e.g: {"op":"set","id":46}
	Op string `json:"op"`
	ID int    `json:"id"`
}
