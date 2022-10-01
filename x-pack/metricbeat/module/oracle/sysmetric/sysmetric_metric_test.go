// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sysmetric

import (
	"testing"
)

func TestSysmetricCollectorCalculateQuery(t *testing.T) {
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
			// Checks if query is generated properly for given array of patterns.
			fields{
				patterns: patterns,
			},
			"SELECT METRIC_NAME, VALUE FROM V$SYSMETRIC WHERE GROUP_ID = 2 AND METRIC_NAME LIKE :pattern0 OR METRIC_NAME LIKE :pattern1 OR METRIC_NAME LIKE :pattern2",
		},
		{
			// Checks if query is generated properly if patterns are not given.
			fields{},
			"SELECT METRIC_NAME, VALUE FROM V$SYSMETRIC WHERE GROUP_ID = 2 AND METRIC_NAME LIKE :pattern0",
		},
	}
	for _, tt := range tests {
		t.Run("test func CalculateQuery()", func(t *testing.T) {
			e := &sysmetricCollector{
				patterns: tt.fields.patterns,
			}
			if got := e.calculateQuery(); got != tt.want {
				t.Errorf("sysmetricCollector.calculateQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
