// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func Test_httpReadJSON(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantObjs       []mapstr.M
		wantStatus     int
		wantErr        bool
		wantRawMessage []json.RawMessage
	}{
		{
			name:       "single object",
			body:       `{"a": 42, "b": "c"}`,
			wantObjs:   []mapstr.M{{"a": int64(42), "b": "c"}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "array accepted",
			body:       `[{"a":"b"},{"c":"d"}]`,
			wantObjs:   []mapstr.M{{"a": "b"}, {"c": "d"}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not an object not accepted",
			body:       `42`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "not an object mixed",
			body: `[{a:1},
								42,
							{a:2}]`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "sequence of objects accepted (CRLF)",
			body:       `{"a":1}` + "\r" + `{"a":2}`,
			wantObjs:   []mapstr.M{{"a": int64(1)}, {"a": int64(2)}},
			wantStatus: http.StatusOK,
		},
		{
			name: "sequence of objects accepted (LF)",
			body: `{"a":"1"}
									{"a":"2"}`,
			wantRawMessage: []json.RawMessage{
				[]byte(`{"a":"1"}`),
				[]byte(`{"a":"2"}`),
			},
			wantObjs:   []mapstr.M{{"a": "1"}, {"a": "2"}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "sequence of objects accepted (SP)",
			body:       `{"a":"2"} {"a":"2"}`,
			wantObjs:   []mapstr.M{{"a": "2"}, {"a": "2"}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "sequence of objects accepted (no separator)",
			body:       `{"a":"2"}{"a":"2"}`,
			wantObjs:   []mapstr.M{{"a": "2"}, {"a": "2"}},
			wantStatus: http.StatusOK,
		},
		{
			name: "not an object in sequence",
			body: `{"a":"2"}
									42
						 {"a":"2"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "array of objects in stream",
			body: `{"a":"1"} [{"a":"2"},{"a":"3"}] {"a":"4"}`,
			wantRawMessage: []json.RawMessage{
				[]byte(`{"a":"1"}`),
				[]byte(`{"a":"2"}`),
				[]byte(`{"a":"3"}`),
				[]byte(`{"a":"4"}`),
			},
			wantObjs:   []mapstr.M{{"a": "1"}, {"a": "2"}, {"a": "3"}, {"a": "4"}},
			wantStatus: http.StatusOK,
		},
		{
			name: "numbers",
			body: `{"a":1} [{"a":false},{"a":3.14}] {"a":-4}`,
			wantRawMessage: []json.RawMessage{
				[]byte(`{"a":1}`),
				[]byte(`{"a":false}`),
				[]byte(`{"a":3.14}`),
				[]byte(`{"a":-4}`),
			},
			wantObjs: []mapstr.M{
				{"a": int64(1)},
				{"a": false},
				{"a": 3.14},
				{"a": int64(-4)},
			},
			wantStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotObjs, rawMessages, gotStatus, err := httpReadJSON(strings.NewReader(tt.body))
			if (err != nil) != tt.wantErr {
				t.Errorf("httpReadJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.EqualValues(t, tt.wantObjs, gotObjs) {
				t.Errorf("httpReadJSON() gotObjs = %v, want %v", gotObjs, tt.wantObjs)
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("httpReadJSON() gotStatus = %v, want %v", gotStatus, tt.wantStatus)
			}
			if tt.wantRawMessage != nil {
				assert.Equal(t, tt.wantRawMessage, rawMessages)
			}
			assert.Equal(t, len(gotObjs), len(rawMessages))
		})
	}
}

type publisher struct {
	mu     sync.Mutex
	events []beat.Event
}

func (p *publisher) Publish(e beat.Event) {
	p.mu.Lock()
	p.events = append(p.events, e)
	p.mu.Unlock()
}

func Test_apiResponse(t *testing.T) {
	testCases := []struct {
		name         string        // Sub-test name.
		conf         config        // Load configuration.
		request      *http.Request // Input request.
		events       []mapstr.M    // Expected output events.
		wantStatus   int           // Expected response code.
		wantResponse string        // Expected response message.
	}{
		{
			name: "single event",
			conf: defaultConfig(),
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"id":0}`))
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
			events: []mapstr.M{
				{
					"json": mapstr.M{
						"id": int64(0),
					},
				},
			},
			wantStatus:   http.StatusOK,
			wantResponse: `{"message": "success"}`,
		},
		{
			name: "single event gzip",
			conf: defaultConfig(),
			request: func() *http.Request {
				buf := new(bytes.Buffer)
				b := gzip.NewWriter(buf)
				_, _ = io.WriteString(b, `{"id":0}`)
				b.Close()

				req := httptest.NewRequest(http.MethodPost, "/", buf)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Content-Encoding", "gzip")
				return req
			}(),
			events: []mapstr.M{
				{
					"json": mapstr.M{
						"id": int64(0),
					},
				},
			},
			wantStatus:   http.StatusOK,
			wantResponse: `{"message": "success"}`,
		},
		{
			name: "multiple events gzip",
			conf: defaultConfig(),
			request: func() *http.Request {
				events := []string{
					`{"id":0}`,
					`{"id":1}`,
				}

				buf := new(bytes.Buffer)
				b := gzip.NewWriter(buf)
				_, _ = io.WriteString(b, strings.Join(events, "\n"))
				b.Close()

				req := httptest.NewRequest(http.MethodPost, "/", buf)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Content-Encoding", "gzip")
				return req
			}(),
			events: []mapstr.M{
				{
					"json": mapstr.M{
						"id": int64(0),
					},
				},
				{
					"json": mapstr.M{
						"id": int64(1),
					},
				},
			},
			wantStatus:   http.StatusOK,
			wantResponse: `{"message": "success"}`,
		},
		{
			name: "validate CRC request",
			conf: config{
				CRCProvider: "Zoom",
				CRCSecret:   "secretValueTest",
			},
			request: func() *http.Request {
				buf := bytes.NewBufferString(
					`{
						"event_ts":1654503849680,
						"event":"endpoint.url_validation",
						"payload": {
							"plainToken":"qgg8vlvZRS6UYooatFL8Aw"
						}
					}`,
				)
				req := httptest.NewRequest(http.MethodPost, "/", buf)
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
			events:       nil,
			wantStatus:   http.StatusOK,
			wantResponse: `{"encryptedToken":"70c1f2e2e6ca2d39297490d1f9142c7d701415ea8e6151f6562a08fa657a40ff","plainToken":"qgg8vlvZRS6UYooatFL8Aw"}`,
		},
		{
			name: "malformed CRC request",
			conf: config{
				CRCProvider: "Zoom",
				CRCSecret:   "secretValueTest",
			},
			request: func() *http.Request {
				buf := bytes.NewBufferString(
					`{
						"event_ts":1654503849680,
						"event":"endpoint.url_validation",
						"payload": {
							"plainToken":"qgg8vlvZRS6UYooatFL8Aw
						}
					}`,
				)
				req := httptest.NewRequest(http.MethodPost, "/", buf)
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
			events:       nil,
			wantStatus:   http.StatusBadRequest,
			wantResponse: `{"message":"malformed JSON object at stream position 0: invalid character '\\n' in string literal"}`,
		},
		{
			name: "empty CRC challenge",
			conf: config{
				CRCProvider: "Zoom",
				CRCSecret:   "secretValueTest",
			},
			request: func() *http.Request {
				buf := bytes.NewBufferString(
					`{
						"event_ts":1654503849680,
						"event":"endpoint.url_validation",
						"payload": {
							"plainToken":""
						}
					}`,
				)
				req := httptest.NewRequest(http.MethodPost, "/", buf)
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
			events:       nil,
			wantStatus:   http.StatusBadRequest,
			wantResponse: `{"message":"failed decoding \"payload.plainToken\" from CRC request"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			pub := new(publisher)
			apiHandler := newHandler(tc.conf, pub, logp.NewLogger("http_endpoint.test"))

			// Execute handler.
			respRec := httptest.NewRecorder()
			apiHandler.ServeHTTP(respRec, tc.request)

			// Validate responses.
			assert.Equal(t, tc.wantStatus, respRec.Code)
			assert.Equal(t, tc.wantResponse, strings.TrimSuffix(respRec.Body.String(), "\n"))
			require.Len(t, pub.events, len(tc.events))

			for i, evt := range pub.events {
				assert.EqualValues(t, tc.events[i], evt.Fields)
			}
		})
	}
}
