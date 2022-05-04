// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sysmetric

import (
	"testing"
)

func TestSysmetricExtractorCalculateQuery(t *testing.T) {
	type fields struct {
		patterns []interface{}
	}
	strpatterns := []string{"foo%", "%bar", "%foobar%"}
	patterns := make([]interface{}, len(strpatterns))
	for i, v := range strpatterns {
		patterns[i] = v
	}
	tests := []struct {
		fields fields
		want   string
	}{
		{
			fields{
				patterns: patterns,
			},
			"SELECT * FROM V$SYSMETRIC WHERE METRIC_NAME LIKE :pattern0 OR METRIC_NAME LIKE :pattern1 OR METRIC_NAME LIKE :pattern2",
		},
		{
			fields{},
			"SELECT * FROM V$SYSMETRIC WHERE METRIC_NAME LIKE :pattern0",
		},
	}
	for _, tt := range tests {
		t.Run("test func CalculateQuery()", func(t *testing.T) {
			e := &sysmetricExtractor{
				patterns: tt.fields.patterns,
			}
			if got := e.calculateQuery(); got != tt.want {
				t.Errorf("sysmetricExtractor.calculateQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
