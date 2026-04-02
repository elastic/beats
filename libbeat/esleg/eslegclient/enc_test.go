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
	"compress/gzip"
	"encoding/json"
	"io"
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

// TestRawEncodingNoDoubleNewline verifies that writing a RawEncoding
// whose bytes already end with '\n' does not produce a double newline
// in the bulk body.  It also verifies that RawEncoding bytes without
// a trailing newline still receive one.
func TestRawEncodingNoDoubleNewline(t *testing.T) {
	// Pre-encode an event via Marshal, which appends a trailing '\n'.
	jsonEnc := NewJSONEncoder(nil, false)
	ev := beat.Event{
		Timestamp: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		Fields:    mapstr.M{"message": "test"},
	}
	require.NoError(t, jsonEnc.Marshal(ev))
	preEncoded := make([]byte, jsonEnc.buf.Len())
	copy(preEncoded, jsonEnc.buf.Bytes())
	require.Equal(t, byte('\n'), preEncoded[len(preEncoded)-1], "pre-encoded event should end with newline")

	meta := map[string]any{"index": map[string]any{"_index": "test"}}

	type encoderCase struct {
		name     string
		newEnc   func(t *testing.T) BodyEncoder
		bodyText func(t *testing.T, enc BodyEncoder) string
	}
	encoders := []encoderCase{
		{
			name:   "JSON encoder",
			newEnc: func(t *testing.T) BodyEncoder { return NewJSONEncoder(nil, false) },
			bodyText: func(t *testing.T, enc BodyEncoder) string {
				body, err := io.ReadAll(enc.Reader())
				require.NoError(t, err)
				return string(body)
			},
		},
		{
			name: "gzip encoder",
			newEnc: func(t *testing.T) BodyEncoder {
				g, err := NewGzipEncoder(5, nil, false)
				require.NoError(t, err)
				return g
			},
			bodyText: func(t *testing.T, enc BodyEncoder) string {
				r, err := gzip.NewReader(enc.Reader())
				require.NoError(t, err)
				defer r.Close()
				body, err := io.ReadAll(r)
				require.NoError(t, err)
				return string(body)
			},
		},
	}

	tests := []struct {
		name     string
		encoding []byte
	}{
		// RawEncoding already has a trailing newline — must not produce "\n\n".
		{"with trailing newline", preEncoded},
		// RawEncoding without a trailing newline — encoder must append one.
		{"without trailing newline", preEncoded[:len(preEncoded)-1]},
	}

	for _, ec := range encoders {
		for _, tc := range tests {
			t.Run(ec.name+"/"+tc.name, func(t *testing.T) {
				enc := ec.newEnc(t)
				require.NoError(t, enc.AddRaw(meta))
				require.NoError(t, enc.AddRaw(RawEncoding{Encoding: tc.encoding}))

				body := ec.bodyText(t, enc)
				assert.NotContains(t, body, "\n\n",
					"bulk body must not contain empty lines; got:\n%s", body)
				lines := splitNDJSON(body)
				assert.Equal(t, 2, len(lines),
					"bulk body should have exactly 2 NDJSON lines (meta + document); got %d:\n%s", len(lines), body)
			})
		}
	}
}

// splitNDJSON splits an NDJSON string into non-empty lines.
func splitNDJSON(s string) []string {
	var lines []string
	for line := range strings.SplitSeq(s, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestEncoderHeaders(t *testing.T) {
	metadata := map[string]any{
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
