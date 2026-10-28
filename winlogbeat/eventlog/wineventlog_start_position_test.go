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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
)

// TestStartAtEndAfterStaleBookmark checks that start_position: end is not
// applied when the reader is reopened after stale bookmark recovery.
func TestStartAtEndAfterStaleBookmark(t *testing.T) {
	writer, teardown := createLog(t)
	defer teardown()
	safeWriteEvent(t, writer, 1, "backlog")

	// A bookmark for a record that does not exist triggers recovery, which
	// clears the bookmark and subscribes from the oldest record.
	stale := checkpoint.EventLogState{
		Bookmark: "<BookmarkList><Bookmark Channel='" + providerName + "' RecordId='1000' IsCurrent='true'/></BookmarkList>",
	}
	log := openLog(t, &stale, map[string]any{
		"name":           providerName,
		"start_position": "end",
		"no_more_events": "stop",
	})
	defer log.Close()
	require.Empty(t, log.Checkpoint().Bookmark, "recovery should have cleared the bookmark")

	// Reopen without reading, as the runner does after a recoverable error.
	require.NoError(t, log.Reset(), "reset must succeed")
	require.NoError(t, log.Open(log.Checkpoint(), nil), "reopen must succeed")

	records, err := log.Read()
	assert.ErrorIs(t, err, io.EOF, "read should drain the channel")
	assert.Len(t, records, 1, "the backlog must be read, not skipped")
}
