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

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestFilestreamMetrics(t *testing.T) {
	env := newInputTestingEnvironment(t)

	testlogName := "test.log"
	inp := env.mustCreateInput(map[string]interface{}{
		"id":                                   "fake-ID",
		"paths":                                []string{env.abspath(testlogName)},
		"prospector.scanner.check_interval":    "24h",
		"close.on_state_change.check_interval": "100ms",
		"close.on_state_change.inactive":       "2s",
	})

	testlines := []byte("first line\nsecond line\nthird line\n")
	env.mustWriteToFile(testlogName, testlines)

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)

	env.waitUntilEventCount(3)
	env.requireOffsetInRegistry(testlogName, "fake-ID", len(testlines))
	env.waitUntilHarvesterIsDone()

	checkMetrics(t, "fake-ID", expectedMetrics{
		FilesOpened:      1,
		FilesClosed:      1,
		FilesActive:      0,
		MessagesRead:     3,
		BytesProcessed:   34,
		EventsProcessed:  3,
		ProcessingErrors: 0,
	})

	cancelInput()
	env.waitUntilInputStops()
}

type expectedMetrics struct {
	FilesOpened      uint64
	FilesClosed      uint64
	FilesActive      uint64
	MessagesRead     uint64
	BytesProcessed   uint64
	EventsProcessed  uint64
	ProcessingErrors uint64
}

func checkMetrics(t *testing.T, id string, expected expectedMetrics) {
	reg := monitoring.GetNamespace("dataset").GetRegistry().Get(id).(*monitoring.Registry)

	require.Equal(t, id, reg.Get("id").(*monitoring.String).Get(), "id")
	require.Equal(t, "filestream", reg.Get("input").(*monitoring.String).Get(), "input")
	require.Equal(t, expected.FilesOpened, reg.Get("files_opened_total").(*monitoring.Uint).Get(), "files_opened_total")
	require.Equal(t, expected.FilesClosed, reg.Get("files_closed_total").(*monitoring.Uint).Get(), "files_closed_total")
	require.Equal(t, expected.MessagesRead, reg.Get("messages_read_total").(*monitoring.Uint).Get(), "messages_read_total")
	require.Equal(t, expected.BytesProcessed, reg.Get("bytes_processed_total").(*monitoring.Uint).Get(), "bytes_processed_total")
	require.Equal(t, expected.EventsProcessed, reg.Get("events_processed_total").(*monitoring.Uint).Get(), "events_processed_total")
	require.Equal(t, expected.ProcessingErrors, reg.Get("processing_errors_total").(*monitoring.Uint).Get(), "processing_errors_total")
}
