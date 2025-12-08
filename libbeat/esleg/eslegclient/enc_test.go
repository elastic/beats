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
	assert.Equal(t, encoder.buf.String(), "{\"@timestamp\":\"2017-11-07T12:00:00.000Z\",\"field1\":\"value1\"}\n",
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
	assert.Equal(t, encoder.buf.String(), "{\"timestamp\":\"2017-11-07T12:00:00.000Z\",\"field1\":\"value1\"}\n",
		"Unexpected marshaled format of report.Event")
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
