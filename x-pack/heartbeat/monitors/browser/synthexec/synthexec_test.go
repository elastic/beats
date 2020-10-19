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

package synthexec

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"
)

func TestRunCmd(t *testing.T) {

}

func TestLineToSynthEventFactory(t *testing.T) {
	testType := "mytype"
	testText := "sometext"
	f := lineToSynthEventFactory(testType)
	res, err := f([]byte(testText), testText)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, testType, res.Type)
	require.Equal(t, testText, res.Payload["message"])
	require.Greater(t, res.TimestampEpochMillis, float64(0))
}

func TestJsonToSynthEvent(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		synthEvent *SynthEvent
		wantErr    bool
	}{
		{
			name:       "an empty line",
			line:       "",
			synthEvent: nil,
		},
		{
			name:       "a blank line",
			line:       "   ",
			synthEvent: nil,
		},
		{
			name:       "an invalid line",
			line:       `{"foo": "bar"}"`,
			synthEvent: nil,
			wantErr:    true,
		},
		{
			name: "a valid line",
			line: `{"@timestamp":7165676811882692608,"type":"step/end","journey":{"name":"inline","id":"inline"},"step":{"name":"Go to home page","index":0},"payload":{"source":"async ({page, params}) => {await page.goto('http://www.elastic.co')}","duration_ms":3472,"url":"https://www.elastic.co/","status":"succeeded"},"url":"https://www.elastic.co/","package_version":"0.0.1"}`,
			synthEvent: &SynthEvent{
				TimestampEpochMillis: 7165676811882692608,
				Type:                 "step/end",
				Journey: &Journey{
					Name: "inline",
					Id:   "inline",
				},
				Step: &Step{
					Name:  "Go to home page",
					Index: 0,
				},
				Payload: map[string]interface{}{
					"source":      "async ({page, params}) => {await page.goto('http://www.elastic.co')}",
					"duration_ms": float64(3472),
					"url":         "https://www.elastic.co/",
					"status":      "succeeded",
				},
				PackageVersion: "0.0.1",
				URL:            "https://www.elastic.co/",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes, err := jsonToSynthEvent([]byte(tt.line), tt.line)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err, "for line %s", tt.line)
			}

			if diff := deep.Equal(gotRes, tt.synthEvent); diff != nil {
				t.Error(diff)
			}
		})
	}
}
