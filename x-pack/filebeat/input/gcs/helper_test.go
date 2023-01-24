// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func Test_httpReadJSON(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantObjs []mapstr.M
		wantErr  bool
	}{
		{
			name:     "single object",
			body:     `{"a": 42, "b": "c"}`,
			wantObjs: []mapstr.M{{"a": int64(42), "b": "c"}},
		},
		{
			name:     "array accepted",
			body:     `[{"a":"b"},{"c":"d"}]`,
			wantObjs: []mapstr.M{{"a": "b"}, {"c": "d"}},
		},
		{
			name:    "not an object not accepted",
			body:    `42`,
			wantErr: true,
		},
		{
			name: "not an object mixed",
			body: `[{a:1},
								42,
							{a:2}]`,
			wantErr: true,
		},
		{
			name:     "sequence of objects accepted (CRLF)",
			body:     `{"a":1}` + "\r" + `{"a":2}`,
			wantObjs: []mapstr.M{{"a": int64(1)}, {"a": int64(2)}},
		},
		{
			name: "sequence of objects accepted (LF)",
			body: `{"a":"1"}
									{"a":"2"}`,
			wantObjs: []mapstr.M{{"a": "1"}, {"a": "2"}},
		},
		{
			name:     "sequence of objects accepted (SP)",
			body:     `{"a":"2"} {"a":"2"}`,
			wantObjs: []mapstr.M{{"a": "2"}, {"a": "2"}},
		},
		{
			name:     "sequence of objects accepted (no separator)",
			body:     `{"a":"2"}{"a":"2"}`,
			wantObjs: []mapstr.M{{"a": "2"}, {"a": "2"}},
		},
		{
			name: "not an object in sequence",
			body: `{"a":"2"}
									42
						 {"a":"2"}`,
			wantErr: true,
		},
		{
			name:     "array of objects in stream",
			body:     `{"a":"1"} [{"a":"2"},{"a":"3"}] {"a":"4"}`,
			wantObjs: []mapstr.M{{"a": "1"}, {"a": "2"}, {"a": "3"}, {"a": "4"}},
		},
		{
			name: "numbers",
			body: `{"a":1} [{"a":false},{"a":3.14}] {"a":-4}`,
			wantObjs: []mapstr.M{
				{"a": int64(1)},
				{"a": false},
				{"a": 3.14},
				{"a": int64(-4)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotObjs, err := decodeJSON(strings.NewReader(tt.body))
			if (err != nil) != tt.wantErr {
				t.Errorf("httpReadJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.EqualValues(t, tt.wantObjs, gotObjs) {
				t.Errorf("httpReadJSON() gotObjs = %v, want %v", gotObjs, tt.wantObjs)
			}
		})
	}
}
