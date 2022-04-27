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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/reader"
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
		Fields:  common.MapStr{},
	}
	t.currentLine++

	return m, nil
}

func TestNewParser(t *testing.T) {
	type testResult struct {
		Timestamp time.Time
		Content   []byte
		Fields    common.MapStr
		WantErr   bool
	}

	referenceTime := time.Now()
	tests := map[string]struct {
		Config Config
		In     [][]byte
		Want   []testResult
	}{
		"format-auto": {
			Config: DefaultConfig(),
			In: [][]byte{
				[]byte(`<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog 1024 ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"][examplePriority@32473 class="high"] this is the message`),
				[]byte(`<13>Oct 11 22:14:15 test-host su[1024]: this is the message`),
				[]byte(`Not a valid message.`),
			},
			Want: []testResult{
				{
					Timestamp: mustParseTime(time.RFC3339Nano, "2003-10-11T22:14:15.003Z", nil),
					Content:   []byte("this is the message"),
					Fields: common.MapStr{
						"log": common.MapStr{
							"syslog": common.MapStr{
								"priority": 165,
								"facility": common.MapStr{
									"code": 20,
									"name": "local4",
								},
								"severity": common.MapStr{
									"code": 5,
									"name": "Notice",
								},
								"hostname": "mymachine.example.com",
								"appname":  "evntslog",
								"procid":   "1024",
								"msgid":    "ID47",
								"version":  "1",
								"structured_data": map[string]map[string]string{
									"examplePriority@32473": {
										"class": "high",
									},
									"exampleSDID@32473": {
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
					Timestamp: mustParseTime(time.Stamp, "Oct 11 22:14:15", time.Local),
					Content:   []byte("this is the message"),
					Fields: common.MapStr{
						"log": common.MapStr{
							"syslog": common.MapStr{
								"priority": 13,
								"facility": common.MapStr{
									"code": 1,
									"name": "user-level",
								},
								"severity": common.MapStr{
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
					Timestamp: referenceTime,
					WantErr:   true,
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
				messages:      tc.In,
				referenceTime: referenceTime,
			}
			parser := NewParser(r, &tc.Config)

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

			assert.Len(t, got, len(tc.Want))
			for i, want := range tc.Want {
				if want.WantErr {
					assert.Equal(t, tc.In[i], got[i].Content)
					assert.Equal(t, len(tc.In[i]), got[i].Bytes)
					assert.Equal(t, referenceTime, got[i].Ts)

					if tc.Config.AddErrorKey {
						_, errMsgErr := got[i].Fields.GetValue("error.message")
						assert.NoError(t, errMsgErr, "Expected error.message when Config.AddErrorKey true")
					}
				} else {
					assert.Equal(t, want.Timestamp, got[i].Ts)
					assert.Equal(t, want.Content, got[i].Content)
					assert.Equal(t, len(want.Content), got[i].Bytes)
					assert.Equal(t, want.Fields, got[i].Fields)
				}
			}
		})
	}
}
