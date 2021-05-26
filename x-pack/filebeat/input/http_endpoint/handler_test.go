// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
)

func Test_httpReadJSON(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantObjs   []common.MapStr
		wantStatus int
		wantErr    bool
	}{
		{
			name:     "single object",
			body:     `{"a": 42, "b": "c"}`,
			wantObjs: []common.MapStr{{"a": float64(42), "b": "c"}},
		},
		{
			name:     "array accepted",
			body:     `[{"a":"b"},{"c":"d"}]`,
			wantObjs: []common.MapStr{{"a": "b"}, {"c": "d"}},
		},
		{
			name:       "not an object not accepted",
			body:       `42`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "not an object mixed",
			body:       "[{\"a\":1},\n42,\n{\"a\":2}]",
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:     "sequence of objects accepted (CRLF)",
			body:     "{\"a\":1}\r\n{\"a\":2}",
			wantObjs: []common.MapStr{{"a": float64(1)}, {"a": float64(2)}},
		},
		{
			name:     "sequence of objects accepted (LF)",
			body:     "{\"a\":1}\n{\"a\":2}",
			wantObjs: []common.MapStr{{"a": float64(1)}, {"a": float64(2)}},
		},
		{
			name:     "sequence of objects accepted (SP)",
			body:     "{\"a\":1} {\"a\":2}",
			wantObjs: []common.MapStr{{"a": float64(1)}, {"a": float64(2)}},
		},
		{
			name:     "sequence of objects accepted (no separator)",
			body:     "{\"a\":1}{\"a\":2}",
			wantObjs: []common.MapStr{{"a": float64(1)}, {"a": float64(2)}},
		},
		{
			name:       "not an object in sequence",
			body:       "{\"a\":1}\n42\n{\"a\":2}",
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:     "array of objects in stream",
			body:     `{"a":1} [{"a":2},{"a":3}] {"a":4}`,
			wantObjs: []common.MapStr{{"a": float64(1)}, {"a": float64(2)}, {"a": float64(3)}, {"a": float64(4)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotObjs, gotStatus, err := httpReadJSON(strings.NewReader(tt.body))
			if (err != nil) != tt.wantErr {
				t.Errorf("httpReadJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotObjs, tt.wantObjs) {
				t.Errorf("httpReadJSON() gotObjs = %v, want %v", gotObjs, tt.wantObjs)
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("httpReadJSON() gotStatus = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}
