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

package eslegclient

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/monitoring/report"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestJSONEncoderMarshalBeatEvent(t *testing.T) {
	encoder := NewJSONEncoder(nil, true)
	event := beat.Event{
		Timestamp: time.Date(2017, time.November, 7, 12, 0, 0, 0, time.UTC),
		Fields: mapstr.M{
			"field1": "value1",
		},
	}

	err := encoder.Marshal(event)
	if err != nil {
		t.Errorf("Error while marshaling beat.Event using JSONEncoder: %v", err)
	}
	assert.JSONEq(t, "{\"@timestamp\":\"2017-11-07T12:00:00.000Z\",\"field1\":\"value1\"}\n", encoder.buf.String(),
		"Unexpected marshaled format of beat.Event")
}

func TestJSONEncoderMarshalMonitoringEvent(t *testing.T) {
	encoder := NewJSONEncoder(nil, true)
	event := report.Event{
		Timestamp: time.Date(2017, time.November, 7, 12, 0, 0, 0, time.UTC),
		Fields: mapstr.M{
			"field1": "value1",
		},
	}

	err := encoder.Marshal(event)
	if err != nil {
		t.Errorf("Error while marshaling report.Event using JSONEncoder: %v", err)
	}
	assert.JSONEq(t, "{\"timestamp\":\"2017-11-07T12:00:00.000Z\",\"field1\":\"value1\"}\n", encoder.buf.String(),
		"Unexpected marshaled format of report.Event")
}

// TestRawEncodingNoDoubleNewline verifies that writing a RawEncoding whose
// bytes already end with '\n' does not produce a double newline in the output
// buffer. A double newline in an NDJSON bulk body creates an empty line that
// Elasticsearch-compatible endpoints (Axiom, OpenSearch, etc.) reject.
func TestRawEncodingNoDoubleNewline(t *testing.T) {
	// Pre-encode an event via Marshal, which appends a trailing '\n'.
	encoder := NewJSONEncoder(nil, false)
	event := beat.Event{
		Timestamp: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		Fields:    mapstr.M{"message": "test"},
	}
	err := encoder.Marshal(event)
	require.NoError(t, err)
	preEncoded := make([]byte, encoder.buf.Len())
	copy(preEncoded, encoder.buf.Bytes())

	// Verify the pre-encoded bytes end with exactly one newline.
	require.True(t, len(preEncoded) > 0 && preEncoded[len(preEncoded)-1] == '\n',
		"pre-encoded event should end with newline")

	// Now simulate a bulk body: meta line + RawEncoding document.
	encoder.Reset()
	meta := map[string]interface{}{
		"index": map[string]interface{}{"_index": "test"},
	}
	err = encoder.AddRaw(meta)
	require.NoError(t, err)
	err = encoder.AddRaw(RawEncoding{Encoding: preEncoded})
	require.NoError(t, err)

	body := encoder.buf.String()

	// The body must not contain "\n\n" (double newline / empty line).
	assert.NotContains(t, body, "\n\n",
		"bulk body must not contain an empty line from double newline; got:\n%s", body)

	// The body should be exactly: meta\ndocument\n
	lines := splitNDJSON(body)
	assert.Equal(t, 2, len(lines),
		"bulk body should have exactly 2 NDJSON lines (meta + document); got %d:\n%s", len(lines), body)
}

// splitNDJSON splits an NDJSON string into non-empty lines.
func splitNDJSON(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestEncoderHeaders(t *testing.T) {
	metadata := map[string]interface{}{
		"something": "important",
	}
	metadataRaw, err := json.Marshal(metadata)
	require.NoError(t, err)

	event := beat.Event{
		Timestamp: time.Date(2017, time.November, 7, 12, 0, 0, 0, time.UTC),
		Fields: mapstr.M{
			"field1": "value1",
		},
	}
	eventRaw, err := json.Marshal(event)
	require.NoError(t, err)

	// + 1 for each \n
	bodyLength := strconv.Itoa(len(metadataRaw) + len(eventRaw) + 2)

	gz, err := NewGzipEncoder(5, nil, false)
	require.NoError(t, err)

	tests := []struct {
		name            string
		encoder         BodyEncoder
		expExtraHeaders map[string]string
	}{
		{
			name:    "JSON encoder",
			encoder: NewJSONEncoder(nil, false),
		},
		{
			name:    "GZIP encoder",
			encoder: gz,
			expExtraHeaders: map[string]string{
				headerContentEncoding: "gzip",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("before reset", func(t *testing.T) {
				err := tc.encoder.AddRaw(metadata)
				require.NoError(t, err)
				err = tc.encoder.AddRaw(RawEncoding{eventRaw})
				require.NoError(t, err)

				actualHeader := make(http.Header)
				tc.encoder.AddHeader(&actualHeader)
				errFormat := "wrong %q header before reset"
				assert.Equal(t, "application/json; charset=UTF-8", actualHeader.Get(headerContentType), errFormat, headerContentType)
				assert.Equal(t, bodyLength, actualHeader.Get(HeaderUncompressedLength), errFormat, HeaderUncompressedLength)
				for name, value := range tc.expExtraHeaders {
					assert.Equal(t, value, actualHeader.Get(name), errFormat, name)
				}
			})

			t.Run("after reset", func(t *testing.T) {
				tc.encoder.Reset()

				errFormat := "wrong %q header after reset"
				actualHeader := make(http.Header)
				tc.encoder.AddHeader(&actualHeader)
				assert.Equal(t, "application/json; charset=UTF-8", actualHeader.Get(headerContentType), errFormat, headerContentType)
				assert.Equal(t, "0", actualHeader.Get(HeaderUncompressedLength), errFormat, HeaderUncompressedLength)
				for name, value := range tc.expExtraHeaders {
					assert.Equal(t, value, actualHeader.Get(name), errFormat, name)
				}
			})

			t.Run("after re-write", func(t *testing.T) {
				err := tc.encoder.AddRaw(metadata)
				require.NoError(t, err)
				err = tc.encoder.AddRaw(RawEncoding{eventRaw})
				require.NoError(t, err)

				errFormat := "wrong %q header after re-write"
				actualHeader := make(http.Header)
				tc.encoder.AddHeader(&actualHeader)
				assert.Equal(t, "application/json; charset=UTF-8", actualHeader.Get(headerContentType), errFormat, headerContentType)
				assert.Equal(t, bodyLength, actualHeader.Get(HeaderUncompressedLength), errFormat, HeaderUncompressedLength)
				for name, value := range tc.expExtraHeaders {
					assert.Equal(t, value, actualHeader.Get(name), "wrong %q header", name)
				}
			})
		})
	}
}
