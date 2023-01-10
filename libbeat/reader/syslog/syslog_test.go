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

package syslog

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var _ reader.Reader = &testReader{}

type testReader struct {
	messages      [][]byte
	currentLine   int
	referenceTime time.Time
}

func (*testReader) Close() error {
	return nil
}

func (t *testReader) Next() (reader.Message, error) {
	if t.currentLine == len(t.messages) {
		return reader.Message{}, io.EOF
	}

	m := reader.Message{
		Ts:      t.referenceTime,
		Content: t.messages[t.currentLine],
		Bytes:   len(t.messages[t.currentLine]),
		Fields:  mapstr.M{},
	}
	t.currentLine++

	return m, nil
}

func TestNewParser(t *testing.T) {
	type testResult struct {
		timestamp time.Time
		content   []byte
		fields    mapstr.M
		wantErr   bool
	}

	referenceTime := time.Now()
	tests := map[string]struct {
		config Config
		in     [][]byte
		want   []testResult
	}{
		"format-auto": {
			config: DefaultConfig(),
			in: [][]byte{
				[]byte(`<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog 1024 ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"] this is the message`),
				[]byte(`<13>Oct 11 22:14:15 test-host su[1024]: this is the message`),
				[]byte(`Not a valid message.`),
			},
			want: []testResult{
				{
					timestamp: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.003Z", nil),
					content:   []byte("this is the message"),
					fields: mapstr.M{
						"log": mapstr.M{
							"syslog": mapstr.M{
								"priority": 165,
								"facility": mapstr.M{
									"code": 20,
									"name": "local4",
								},
								"severity": mapstr.M{
									"code": 5,
									"name": "Notice",
								},
								"hostname": "mymachine.example.com",
								"appname":  "evntslog",
								"procid":   "1024",
								"msgid":    "ID47",
								"version":  "1",
								"structured_data": map[string]interface{}{
									"examplePriority@32473": map[string]interface{}{
										"class": "high",
									},
									"exampleSDID@32473": map[string]interface{}{
										"eventID":     "1011",
										"eventSource": "Application",
										"iut":         "3",
									},
								},
							},
						},
						"message": "this is the message",
					},
				},
				{
					timestamp: mustParseTime(time.Stamp, "Oct 11 22:14:15", time.Local),
					content:   []byte("this is the message"),
					fields: mapstr.M{
						"log": mapstr.M{
							"syslog": mapstr.M{
								"priority": 13,
								"facility": mapstr.M{
									"code": 1,
									"name": "user-level",
								},
								"severity": mapstr.M{
									"code": 5,
									"name": "Notice",
								},
								"hostname": "test-host",
								"appname":  "su",
								"procid":   "1024",
							},
						},
						"message": "this is the message",
					},
				},
				{
					timestamp: referenceTime,
					wantErr:   true,
				},
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got []reader.Message

			r := &testReader{
				messages:      tc.in,
				referenceTime: referenceTime,
			}
			parser := NewParser(r, &tc.config)

			var err error
			var msg reader.Message
			for {
				msg, err = parser.Next()
				if errors.Is(err, io.EOF) {
					break
				}
				assert.NoError(t, err)
				got = append(got, msg)
			}

			assert.Len(t, got, len(tc.want))
			for i, want := range tc.want {
				if want.wantErr {
					assert.Equal(t, tc.in[i], got[i].Content)
					assert.Equal(t, len(tc.in[i]), got[i].Bytes)
					assert.Equal(t, referenceTime, got[i].Ts)

					if tc.config.AddErrorKey {
						_, errMsgErr := got[i].Fields.GetValue("error.message")
						assert.NoError(t, errMsgErr, "Expected error.message when Config.AddErrorKey true")
					}
				} else {
					assert.Equal(t, want.timestamp, got[i].Ts)
					assert.Equal(t, want.content, got[i].Content)
					assert.Equal(t, len(want.content), got[i].Bytes)
					assert.Equal(t, want.fields, got[i].Fields)
				}
			}
		})
	}
}
