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
	"context"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestFilestreamMetrics(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	id := uuid.Must(uuid.NewV4()).String()
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                     id,
		"paths":                                  []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":      "24h",
		"close.on_state_change.check_interval":   "100ms",
		"close.on_state_change.inactive":         "2s",
		"prospector.scanner.fingerprint.enabled": false,
		"file_identity.native":                   map[string]any{},
		"message_max_bytes":                      20,
		"parsers": []map[string]interface{}{
			{
				"multiline": map[string]interface{}{
					"type":      "pattern",
					"pattern":   "^multiline",
					"negate":    true,
					"match":     "after",
					"max_lines": 1,
					"timeout":   "1s",
				},
			},
		},
	})

	testlines := []byte("first line\nsecond line\nthird line\nthis is a very long line exceeding message_max_bytes\nmultiline first line\nmultiline second line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, id, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, id, len(testlines))
	env.waitUntilHarvesterIsDone()

	checkMetrics(t, env.monitoring, id, expectedMetrics{
		FilesOpened:       1,
		FilesClosed:       1,
		FilesActive:       0,
		MessagesRead:      3,
		MessagesTruncated: 2,
		BytesProcessed:    130,
		EventsProcessed:   3,
		ProcessingErrors:  0,
	})

	cancelInput()
	env.waitUntilInputStops()
}

func TestFilestreamMessageMaxBytesTruncatedMetric(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	id := uuid.Must(uuid.NewV4()).String()
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                     id,
		"paths":                                  []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":      "24h",
		"close.on_state_change.check_interval":   "100ms",
		"close.on_state_change.inactive":         "2s",
		"prospector.scanner.fingerprint.enabled": false,
		"file_identity.native":                   map[string]any{},
		"message_max_bytes":                      20,
	})

	testlines := []byte("first line\nsecond line\nthird line\nthis is a long line exceeding message_max_bytes\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, id, inp)

	env.waitUntilEventCount(4)
	env.requireOffsetInRegistry(testlogName, id, len(testlines))
	env.waitUntilHarvesterIsDone()

	checkMetrics(t, env.monitoring, id, expectedMetrics{
		FilesOpened:       1,
		FilesClosed:       1,
		FilesActive:       0,
		MessagesRead:      4,
		MessagesTruncated: 1,
		BytesProcessed:    82,
		EventsProcessed:   4,
		ProcessingErrors:  0,
	})

	cancelInput()
	env.waitUntilInputStops()
}

func TestFilestreamMultilineMaxLinesTruncatedMetric(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	id := uuid.Must(uuid.NewV4()).String()
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                     id,
		"paths":                                  []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":      "24h",
		"close.on_state_change.check_interval":   "100ms",
		"close.on_state_change.inactive":         "2s",
		"prospector.scanner.fingerprint.enabled": false,
		"file_identity.native":                   map[string]any{},
		"parsers": []map[string]interface{}{
			{
				"multiline": map[string]interface{}{
					"type":      "pattern",
					"pattern":   "^multiline",
					"negate":    true,
					"match":     "after",
					"max_lines": 1,
					"timeout":   "1s",
				},
			},
		},
	})

	testlines := []byte("first line\nsecond line\nmultiline first line\nmultiline second line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, id, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, id, len(testlines))
	env.waitUntilHarvesterIsDone()

	checkMetrics(t, env.monitoring, id, expectedMetrics{
		FilesOpened:       1,
		FilesClosed:       1,
		FilesActive:       0,
		MessagesRead:      3,
		MessagesTruncated: 1,
		BytesProcessed:    66,
		EventsProcessed:   3,
		ProcessingErrors:  0,
	})

	cancelInput()
	env.waitUntilInputStops()
}

type expectedMetrics struct {
	FilesOpened       uint64
	FilesClosed       uint64
	FilesActive       uint64
	MessagesRead      uint64
	MessagesTruncated uint64
	BytesProcessed    uint64
	EventsProcessed   uint64
	ProcessingErrors  uint64
}

func checkMetrics(t *testing.T, mon beat.Monitoring, id string, expected expectedMetrics) {
	reg, ok := mon.InputsRegistry().Get(id).(*monitoring.Registry)
	require.True(t, ok, "registry not found")

	require.Equal(t, id, reg.Get("id").(*monitoring.String).Get(), "id")
	require.Equal(t, "filestream", reg.Get("input").(*monitoring.String).Get(), "input")
	require.Equal(t, expected.FilesOpened, reg.Get("files_opened_total").(*monitoring.Uint).Get(), "files_opened_total")
	require.Equal(t, expected.FilesClosed, reg.Get("files_closed_total").(*monitoring.Uint).Get(), "files_closed_total")
	require.Equal(t, expected.MessagesRead, reg.Get("messages_read_total").(*monitoring.Uint).Get(), "messages_read_total")
	require.Equal(t, expected.MessagesTruncated, reg.Get("messages_truncated_total").(*monitoring.Uint).Get(), "messages_truncated_total")
	require.Equal(t, expected.BytesProcessed, reg.Get("bytes_processed_total").(*monitoring.Uint).Get(), "bytes_processed_total")
	require.Equal(t, expected.EventsProcessed, reg.Get("events_processed_total").(*monitoring.Uint).Get(), "events_processed_total")
	require.Equal(t, expected.ProcessingErrors, reg.Get("processing_errors_total").(*monitoring.Uint).Get(), "processing_errors_total")
}
