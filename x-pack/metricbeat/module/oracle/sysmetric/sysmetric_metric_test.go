// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sysmetric

import (
	"testing"
)

func TestSysmetricExtractorCalQuery(t *testing.T) {
	type fields struct {
		patterns []string
	}
	tests := []struct {
		fields fields
		want   string
	}{
		{
			fields{
				patterns: []string{"foo%", "%bar", "%foobar%"},
			},
			"SELECT * FROM V$SYSMETRIC WHERE (METRIC_NAME LIKE 'foo%' OR METRIC_NAME LIKE '%bar' OR METRIC_NAME LIKE '%foobar%')",
		},
		{
			fields{},
			"SELECT * FROM V$SYSMETRIC WHERE (METRIC_NAME LIKE '%')",
		},
	}
	for _, tt := range tests {
		t.Run("test func CalQuery()", func(t *testing.T) {
			e := &sysmetricExtractor{
				patterns: tt.fields.patterns,
			}
			if got := e.calQuery(); got != tt.want {
				t.Errorf("sysmetricExtractor.calQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
