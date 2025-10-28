// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package tables_test

import(
	"time"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/tables"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists/testdata"
	"testing"
	"reflect"
)

func TestJumpListEntry_ToMap(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		have    tables.JumpListEntry
		want    map[string]string
		wantErr bool
	}{
		{
			name: "test_to_map",
			have: tables.JumpListEntry{
				LinkPath: "test_value",
				TargetCreatedTime: testdata.GetPredictableTime(1),
				TargetModifiedTime: testdata.GetPredictableTime(2),
				TargetAccessedTime: testdata.GetPredictableTime(3),
				TargetSize: 100,
				TargetPath: "test_value",
			},
			want: map[string]string{
				"link_path": "test_value",
				"target_created_time": testdata.GetPredictableTime(1).Format(time.RFC3339),
				"target_modified_time": testdata.GetPredictableTime(2).Format(time.RFC3339),
				"target_accessed_time": testdata.GetPredictableTime(3).Format(time.RFC3339),
				"target_size": "100",
				"target_path": "test_value",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := tt.have.ToMap()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ToMap() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ToMap() succeeded unexpectedly")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
